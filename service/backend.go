package service

const (
	// Router paths
	pathStatus       = "/status"
	pathCreateWallet = "/wallet"
	pathGetWallets   = "/wallets"
	pathGetSignature = "/sign"

	// To change these parameters, you must first delete the text fixture files in test/_fixtures/ and then run the keygen test alone.
	// Then the signing and resharing tests will work with the new n, t configuration using the newly written fixture files.
	Participants = 5
	Threshold    = Participants / 2

	TestFixtureDirFormat  = "%s/_ecdsa_fixtures"
	TestFixtureFileFormat = "keygen_data_%d.json"
)
