package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/test"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/crypto"
)

func generateTSSKey() (keygen.LocalPartySaveData, error) {
	fmt.Println("generate TSS Key")
	// preParams, _ := keygen.GeneratePreParams(1 * time.Minute)
	parties := make([]*keygen.LocalParty, 0, Participants)
	fixtures, partyIDs, err := keygen.LoadKeygenTestFixtures(Participants)
	fmt.Println(len(fixtures))
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
		partyIDs = tss.GenerateTestPartyIDs(Participants)
	}

	p2pCtx := tss.NewPeerContext(partyIDs)
	errCh := make(chan *tss.Error, len(partyIDs))
	outCh := make(chan tss.Message, len(partyIDs))
	endCh := make(chan *keygen.LocalPartySaveData, len(partyIDs))

	updater := test.SharedPartyUpdater

	for i := 0; i < Participants; i++ {
		var P *keygen.LocalParty
		params := tss.NewParameters(tss.S256(), p2pCtx, partyIDs[i], len(partyIDs), Threshold)
		// if i < len(fixtures) {
		// 	fmt.Println("case 1")
		// 	P = keygen.NewLocalParty(params, outCh, endCh, fixtures[i].LocalPreParams).(*keygen.LocalParty)
		// } else {
		fmt.Println("case 2")
		P = keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)
		// }
		parties = append(parties, P)
		go func(p *keygen.LocalParty) {
			if err := p.Start(); err != nil {
				fmt.Println("Error while starting P")
				errCh <- err
			}
		}(P)
	}

	newKeys := make([]keygen.LocalPartySaveData, Participants)
	partyDone := false
	completedCount := 0
	var wg sync.WaitGroup
	wg.Add(1)

	go func() error {
		defer wg.Done()
		for {
			fmt.Println("FOR")
			select {
			case msg := <-outCh:
				dest := msg.GetTo()
				if dest == nil { // broadcast!
					for _, P := range parties {
						if P.PartyID().Index == msg.GetFrom().Index {
							continue
						}
						go updater(P, msg, errCh)
					}
				} else { // point-to-point!
					if dest[0].Index == msg.GetFrom().Index {
						break
					}
					go updater(parties[dest[0].Index], msg, errCh)
				}
				// `msg` is from one party and needs to be delivered to other parties.
				// In a real scenario, you would identify which parties should receive it.
				// For a broadcast message, deliver it to all other parties except the sender.

				// for _, P := range parties {
				// 	if P.PartyID() == msg.GetFrom() {
				// 		continue // Don't send back to sender
				// 	}
				// 	bz, _, err := msg.WireBytes()
				// 	if err != nil {
				// 		errCh <- P.WrapError(err)
				// 		return err
				// 	}
				// 	pMsg, err := tss.ParseWireMessage(bz, msg.GetFrom(), msg.IsBroadcast())
				// 	if err != nil {
				// 		errCh <- P.WrapError(err)
				// 		return err
				// 	}
				// 	if _, err := P.Update(pMsg); err != nil {
				// 		errCh <- err
				// 	}
				// }

			case save := <-endCh:
				index, err := save.OriginalIndex()
				if err != nil {
					break
				}
				newKeys[index] = *save
				completedCount++
				if completedCount == Participants {
					partyDone = true
				}

			case err := <-errCh:
				fmt.Println("SELECT -3")
				fmt.Println(err)
				return err

			default:
				time.Sleep(10 * time.Second)
			}
			if partyDone {
				break
			}
		}
		return nil
	}()

	wg.Wait()

	fmt.Println("POST FOR")
	return newKeys[0], nil
}

func toAddress(pubKey ecdsa.PublicKey) string {
	uncompressed := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	hash := crypto.Keccak256(uncompressed[1:]) // Remove the 0x04 prefix
	addr := hash[12:]
	return "0x" + hex.EncodeToString(addr)
}
