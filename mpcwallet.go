package main

import (
	"fmt"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/mpcwallet/service"
)

// "github.com/bnb-chain/tss-lib/v2/common"
// "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
// "github.com/bnb-chain/tss-lib/v2/tss"
// "github.com/mpcwallet/service"

func main() {
	// var log = logrus.NewEntry(logrus.New())
	// s, err := service.NewWalletMPCService(service.ServerOpts{ListenAddr: ":8080", Log: log})
	// if err != nil {
	// 	fmt.Printf("Failed to start server: %s", err)
	// }
	// fmt.Println("Server started")
	// s.StartHTTPServer()

	preParams, _ := keygen.GeneratePreParams(1 * time.Minute)
	fixtures, partyIDs, err := keygen.LoadKeygenTestFixtures(service.Participants)
	fmt.Println(len(fixtures))
	if err != nil {
		common.Logger.Info("No test fixtures were found, so the safe primes will be generated from scratch. This may take a while...")
		partyIDs = tss.GenerateTestPartyIDs(service.Participants)
	}
	p2pCtx := tss.NewPeerContext(partyIDs)

	parties := make([]*keygen.LocalParty, service.Participants)
	errCh := make(chan *tss.Error, len(partyIDs))
	outCh := make(chan tss.Message, len(partyIDs))
	endCh := make(chan *keygen.LocalPartySaveData, len(partyIDs))

	// updater := test.SharedPartyUpdater

	// startGR := runtime.NumGoroutine()

	for i := 0; i < service.Participants; i++ {
		var P *keygen.LocalParty
		params := tss.NewParameters(tss.S256(), p2pCtx, partyIDs[i], len(partyIDs), service.Threshold)
		if i < len(fixtures) {
			fmt.Println("case 1")
			P = keygen.NewLocalParty(params, outCh, endCh, fixtures[i].LocalPreParams).(*keygen.LocalParty)
		} else {
			fmt.Println("case 2")
			fmt.Println(params)
			P = keygen.NewLocalParty(params, outCh, endCh, *preParams).(*keygen.LocalParty)
		}
		fmt.Println(P)
		parties = append(parties, P)
		go func(P *keygen.LocalParty) {
			if err := P.Start(); err != nil {
				errCh <- err
			}
			// fmt.Printf("Local Party %s started\n", P.PartyID().Id)
		}(P)
		// partyIDMap[parties[i].PartyID().Id] = parties[i].PartyID()
		fmt.Println(parties)
	}
}

// 	// PHASE: keygen
// 	var ended int32
// keygen:
// 	for {
// 		fmt.Printf("ACTIVE GOROUTINES: %d\n", runtime.NumGoroutine())
// 		select {
// 		case err := <-errCh:
// 			common.Logger.Errorf("Error: %s", err)
// 			break keygen

// 		case msg := <-outCh:
// 			dest := msg.GetTo()
// 			if dest == nil { // broadcast!
// 				for _, P := range parties {
// 					if P.PartyID().Index == msg.GetFrom().Index {
// 						continue
// 					}
// 					go updater(P, msg, errCh)
// 				}
// 			} else { // point-to-point!
// 				if dest[0].Index == msg.GetFrom().Index {
// 					fmt.Printf("party %d tried to send a message to itself (%d)", dest[0].Index, msg.GetFrom().Index)
// 					return
// 				}
// 				go updater(parties[dest[0].Index], msg, errCh)
// 			}

// 		case save := <-endCh:
// 			// SAVE a test fixture file for this P (if it doesn't already exist)
// 			// .. here comes a workaround to recover this party's index (it was removed from save data)
// 			index, err := save.OriginalIndex()
// 			if err != nil {
// 				return
// 			}
// 			tryWriteTestFixtureFile(index, *save)

// 			atomic.AddInt32(&ended, 1)
// 			if atomic.LoadInt32(&ended) == int32(len(partyIDs)) {
// 				fmt.Printf("Done. Received save data from %d participants", ended)

// 				// combine shares for each Pj to get u
// 				u := new(big.Int)
// 				// for j, Pj := range parties {
// 				// 	pShares := make(vss.Shares, 0)
// 				// 	for _, P := range parties {
// 				// 		vssMsgs := P.temp.kgRound2Message1s
// 				// 		share := vssMsgs[j].Content().(*KGRound2Message1).Share
// 				// 		shareStruct := &vss.Share{
// 				// 			Threshold: service.Threshold,
// 				// 			ID:        P.PartyID().KeyInt(),
// 				// 			Share:     new(big.Int).SetBytes(share),
// 				// 		}
// 				// 		pShares = append(pShares, shareStruct)
// 				// 	}
// 				// 	uj, err := pShares[:service.Threshold+1].ReConstruct(tss.S256())
// 				// 	if err != nil {
// 				// 		return
// 				// 	}
// 				// 	uG := crypto.ScalarBaseMult(tss.EC(), uj)
// 				// 	// xj tests: BigXj == xj*G
// 				// 	xj := Pj.data.Xi
// 				// 	gXj := crypto.ScalarBaseMult(tss.EC(), xj)
// 				// 	BigXj := Pj.data.BigXj[j]

// 				// 	// fails if threshold cannot be satisfied (bad share)
// 				// 	{
// 				// 		badShares := pShares[:service.Threshold]
// 				// 		badShares[len(badShares)-1].Share.Set(big.NewInt(0))
// 				// 		_, err := pShares[:service.Threshold].ReConstruct(tss.S256())
// 				// 		if err != nil {
// 				// 			return
// 				// 		}
// 				// 	}
// 				// 	u = new(big.Int).Add(u, uj)
// 				// }

// 				// build ecdsa key pair
// 				pkX, pkY := save.ECDSAPub.X(), save.ECDSAPub.Y()
// 				pk := ecdsa.PublicKey{
// 					Curve: tss.EC(),
// 					X:     pkX,
// 					Y:     pkY,
// 				}
// 				sk := ecdsa.PrivateKey{
// 					PublicKey: pk,
// 					D:         u,
// 				}

// 				fmt.Printf("Public key distribution test done.")

// 				// test sign/verify
// 				data := make([]byte, 32)
// 				for i := range data {
// 					data[i] = byte(i)
// 				}
// 				r, s, err := ecdsa.Sign(rand.Reader, &sk, data)
// 				if err != nil {
// 					return
// 				}
// 				ok := ecdsa.Verify(&pk, data, r, s)
// 				fmt.Printf("ECDSA signing test done. %b", ok)

// 				fmt.Printf("Start goroutines: %d, End goroutines: %d", startGR, runtime.NumGoroutine())

// 				break keygen
// 			}
// 		}
// 	}
// }

// func tryWriteTestFixtureFile(index int, data keygen.LocalPartySaveData) error {
// 	fixtureFileName := makeTestFixtureFilePath(index)

// 	// fixture file does not already exist?
// 	// if it does, we won't re-create it here
// 	fi, err := os.Stat(fixtureFileName)
// 	if !(err == nil && fi != nil && !fi.IsDir()) {
// 		fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
// 		if err != nil {
// 			return fmt.Errorf("unable to open fixture file %s for writing: %s", fixtureFileName, err)
// 		}
// 		bz, err := json.Marshal(&data)
// 		if err != nil {
// 			return fmt.Errorf("unable to marshal save data for fixture file %s: %s", fixtureFileName, err)
// 		}
// 		_, err = fd.Write(bz)
// 		if err != nil {
// 			return fmt.Errorf("unable to write to fixture file %s: %s", fixtureFileName, err)
// 		}
// 		fmt.Printf("Saved a test fixture file for party %d: %s", index, fixtureFileName)
// 	} else {
// 		fmt.Printf("Fixture file already exists for party %d; not re-creating: %s", index, fixtureFileName)
// 	}
// 	return nil
// }

// func makeTestFixtureFilePath(partyIndex int) string {
// 	_, callerFileName, _, _ := runtime.Caller(0)
// 	srcDirName := filepath.Dir(callerFileName)
// 	fixtureDirName := fmt.Sprintf(service.TestFixtureDirFormat, srcDirName)
// 	return fmt.Sprintf("%s/"+service.TestFixtureFileFormat, fixtureDirName, partyIndex)
// }
