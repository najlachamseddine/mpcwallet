package service

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/test"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/ethereum/go-ethereum/crypto"
)

func generateTSSKey() ([]*keygen.LocalPartySaveData, error) {
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
	startGR := runtime.NumGoroutine()

	for i := 0; i < Participants; i++ {
		var P *keygen.LocalParty
		params := tss.NewParameters(tss.S256(), p2pCtx, partyIDs[i], len(partyIDs), Threshold)
		if i < len(fixtures) {
			P = keygen.NewLocalParty(params, outCh, endCh, fixtures[i].LocalPreParams).(*keygen.LocalParty)
		} else {
			P = keygen.NewLocalParty(params, outCh, endCh).(*keygen.LocalParty)
		}
		parties = append(parties, P)
		go func(p *keygen.LocalParty) {
			if err := p.Start(); err != nil {
				fmt.Println("Error while starting P")
				errCh <- err
			}
		}(P)
	}

	newKeys := make([]*keygen.LocalPartySaveData, Participants)
	partyDone := false
	completedCount := 0
	var wg sync.WaitGroup
	wg.Add(1)

	go func() error {
		defer wg.Done()
	keygen:
		for {
			select {
			case msg := <-outCh:
				dest := msg.GetTo()
				if dest == nil {
					for _, P := range parties {
						if P.PartyID().Index == msg.GetFrom().Index {
							continue
						}
						go updater(P, msg, errCh)
					}
				} else {
					if dest[0].Index == msg.GetFrom().Index {
						break
					}
					go updater(parties[dest[0].Index], msg, errCh)
				}

			case save := <-endCh:
				index, err := save.OriginalIndex()
				if err != nil {
					break
				}
				err = tryWriteTestFixtureFile(index, *save)
				if err != nil {
					common.Logger.Errorf("error saving into file %s", err)
					break
				}
				newKeys[index] = save
				completedCount++
				if completedCount == Participants {
					partyDone = true
				}

			case err := <-errCh:
				common.Logger.Errorf("Error: %s", err)
				return err

			default:
				time.Sleep(10 * time.Second)
			}
			if partyDone {
				common.Logger.Infof("Start goroutines: %d, End goroutines: %d", startGR, runtime.NumGoroutine())
				break keygen
			}
		}
		return nil
	}()

	wg.Wait()
	return newKeys, nil
}

func tryWriteTestFixtureFile(index int, data keygen.LocalPartySaveData) error {
	fixtureFileName := makeTestFixtureFilePath(index)
	fi, err := os.Stat(fixtureFileName)
	if !(err == nil && fi != nil && !fi.IsDir()) {
		fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			return err
		}
		defer fd.Close()
		bz, err := json.Marshal(&data)
		if err != nil {
			return err
		}
		_, err = fd.Write(bz)
		if err != nil {
			return err
		}
		fmt.Printf("Saved a test fixture file for party %d: %s", index, fixtureFileName)
	} else {
		fmt.Printf("Fixture file already exists for party %d; not re-creating: %s", index, fixtureFileName)
	}
	return nil
}

func makeTestFixtureFilePath(partyIndex int) string {
	_, callerFileName, _, _ := runtime.Caller(0)
	srcDirName := filepath.Dir(callerFileName)
	fixtureDirName := fmt.Sprintf(TestFixtureDirFormat, srcDirName)
	return fmt.Sprintf("%s/"+TestFixtureFileFormat, fixtureDirName, partyIndex)
}

func toAddress(pubKey ecdsa.PublicKey) string {
	uncompressed := crypto.CompressPubkey(&pubKey)
	hash := crypto.Keccak256(uncompressed[1:]) // Remove the 0x04 prefix
	addr := hash[12:]
	address := "0x" + hex.EncodeToString(addr)
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	if re.MatchString(address) {
		return address
	}
	return ""
}

func tssSign(keysData []*keygen.LocalPartySaveData, msg []byte) (*common.SignatureData, error) {
	partyIDs := make(tss.UnSortedPartyIDs, Threshold+1)
	plucked := make(map[int]interface{}, Threshold+1)
	for i := 0; len(plucked) < Threshold+1; i = (i + 1) % Participants {
		_, have := plucked[i]
		if pluck := rand.Float32() < 0.5; !have && pluck {
			plucked[i] = new(struct{})
		}
	}
	j := 0
	for i := range plucked {
		key := keysData[j]
		pMoniker := fmt.Sprintf("%d", i+1)
		partyIDs[j] = tss.NewPartyID(pMoniker, pMoniker, key.ShareID)
		j++
	}

	signPIDs := tss.SortPartyIDs(partyIDs)
	sort.Slice(keysData, func(i, j int) bool { return keysData[i].ShareID.Cmp(keysData[j].ShareID) == -1 })
	p2pCtx := tss.NewPeerContext(signPIDs)
	parties := make([]*signing.LocalParty, 0, len(signPIDs))

	errCh := make(chan *tss.Error, len(signPIDs))
	outCh := make(chan tss.Message, len(signPIDs))
	endCh := make(chan *common.SignatureData, len(signPIDs))

	updater := test.SharedPartyUpdater

	for i := 0; i < len(signPIDs); i++ {
		params := tss.NewParameters(tss.S256(), p2pCtx, signPIDs[i], len(signPIDs), Threshold)
		P := signing.NewLocalParty(bytesToBigInt(ethereumMessageHash(msg)), params, *keysData[i], outCh, endCh).(*signing.LocalParty)
		parties = append(parties, P)
		go func(P *signing.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
		}(P)
	}

	var ended int32
	var signature *common.SignatureData
	var wg sync.WaitGroup
	wg.Add(1)

	go func() error {
		defer wg.Done()
	signing:
		for {
			select {
			case msg := <-outCh:
				dest := msg.GetTo()
				if dest == nil {
					for _, P := range parties {
						if P.PartyID().Index == msg.GetFrom().Index {
							continue
						}
						go updater(P, msg, errCh)
					}
				} else {
					if dest[0].Index == msg.GetFrom().Index {
						return fmt.Errorf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
					}
					go updater(parties[dest[0].Index], msg, errCh)
				}
			case sig := <-endCh:
				atomic.AddInt32(&ended, 1)
				if atomic.LoadInt32(&ended) == int32(len(signPIDs)) {
					signature = sig
					common.Logger.Info("ECDSA signing test done.")
					break signing
				}
			case err := <-errCh:
				common.Logger.Errorf("Error: %s", err)
				return err
			}
		}
		return nil
	}()

	wg.Wait()
	common.Logger.Infof("POST FOR %s", signature)
	return signature, nil
}

func ethereumMessageHash(message []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	prefixedMsg := []byte(prefix)
	prefixedMsg = append(prefixedMsg, message...)
	hash := crypto.Keccak256(prefixedMsg)
	return hash
}

func bytesToBigInt(data []byte) *big.Int {
	x := new(big.Int)
	x.SetBytes(data)
	return x
}

func verifySignature(pk ecdsa.PublicKey, data []byte, r *big.Int, s *big.Int) bool {
	return ecdsa.Verify(&pk, data, r, s)
}
