// Copyright 2018 Shift Devices AG
// Copyright 2023 Shift Crypto AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lightning

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ArkLabsHQ/fulmine/pkg/boltz"
	"github.com/ArkLabsHQ/fulmine/pkg/swap"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts/types"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/config"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/keystore"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/rates"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/util"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/logging"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/observable"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/observable/action"
	arksdk "github.com/arkade-os/go-sdk"
	grpcclient "github.com/arkade-os/go-sdk/client/grpc"
	indexertransport "github.com/arkade-os/go-sdk/indexer/grpc"
	store "github.com/arkade-os/go-sdk/store"
	arktypes "github.com/arkade-os/go-sdk/types"
	"github.com/breez/breez-sdk-go/breez_sdk"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
)

const (
	breezApiKeyUrl    = "https://bitboxapp.shiftcrypto.io/lightning/breez-api-key"
	greenLightCertUrl = "https://bitboxapp.shiftcrypto.io/lightning/greenlight.crt"
	greenLightKeyUrl  = "https://bitboxapp.shiftcrypto.io/lightning/greenlight-key.pem"
)

// Lightning manages the Breez SDK lightning node.
type Lightning struct {
	observable.Implementation

	backendConfig      *config.Config
	cacheDirectoryPath string
	getKeystore        func() keystore.Keystore
	synced             bool

	log          *logrus.Entry
	sdkService   *breez_sdk.BlockingBreezServices
	httpClient   *http.Client
	ratesUpdater *rates.RateUpdater
	btcCoin      coin.Coin
	arkClient    *arksdk.ArkClient
	boltzApi     *boltz.Api
	swapHandler  *swap.SwapHandler
}

func setupFileBasedArkClient(seed string, dirPath string) (arksdk.ArkClient, error) {
	storeSvc, err := store.NewStore(store.Config{
		ConfigStoreType:  arktypes.FileStore,
		AppDataStoreType: arktypes.SQLStore,
		BaseDir:          path.Join(dirPath, "ark"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup file store: %s", err)
	}

	client, err := arksdk.NewArkClient(storeSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to setup ark client: %s", err)
	}

	if err := client.Init(context.Background(), arksdk.InitArgs{
		WalletType: arksdk.SingleKeyWallet,
		ClientType: arksdk.GrpcClient,
		ServerUrl:  "https://arkade.computer",
		// ServerUrl:  "https://bitcoin-beta.arkade.sh",
		Password: "password",
		Seed:     seed,
		// ExplorerURL:          "https://mempool.arkade.sh", // Cannot GET /api/v1/address/bc1pfw5reg8tymgrm3dundk42g843xqqwh043nlfsuyy3mszfevpl2fsr424tj/utxo
		// ExplorerPollInterval: time.Second * 30, //useless without another param
		WithTransactionFeed: true, //stateful mode
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize wallet: %s", err)
	}
	// client.IsSynced()

	return client, nil
}

// NewLightning creates a new instance of the Lightning struct.
func NewLightning(config *config.Config,
	cacheDirectoryPath string,
	getKeystore func() keystore.Keystore,
	httpClient *http.Client,
	ratesUpdater *rates.RateUpdater,
	btcCoin coin.Coin) *Lightning {
	return &Lightning{
		backendConfig:      config,
		cacheDirectoryPath: cacheDirectoryPath,
		getKeystore:        getKeystore,
		log:                logging.Get().WithGroup("lightning"),
		synced:             false,
		httpClient:         httpClient,
		ratesUpdater:       ratesUpdater,
		btcCoin:            btcCoin,
	}
}

// Activate first creates a mnemonic from the keystore entropy then connects to instance.
func (lightning *Lightning) Activate() error {
	a := boltz.Api{}
	logrus.Info(a)

	lightningConfig := lightning.backendConfig.LightningConfig()

	if len(lightningConfig.Accounts) > 0 {
		return errp.New("Lightning accounts already configured")
	}

	keystore := lightning.getKeystore()
	if keystore == nil || !keystore.SupportsDeterministicEntropy() {
		return errp.New("No keystore available, or firmware out of date")
	}

	entropy, err := keystore.DeterministicEntropy()
	if err != nil {
		return err
	}

	fingerprint, err := keystore.RootFingerprint()
	if err != nil {
		return err
	}

	entropyMnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		lightning.log.WithError(err).Warn("Error generating mnemonic")
		return errp.New("Error generating mnemonic")
	}

	lightningAccount := config.LightningAccountConfig{
		Mnemonic:        entropyMnemonic,
		RootFingerprint: fingerprint,
		Code:            types.Code(strings.Join([]string{"v0-", hex.EncodeToString(fingerprint), "-ln-0"}, "")),
		Number:          0,
	}
	lightningConfig.Accounts = append(lightningConfig.Accounts, &lightningAccount)

	if err = lightning.setLightningConfig(lightningConfig); err != nil {
		return err
	}

	arkClient, err := setupFileBasedArkClient(hex.EncodeToString(entropy), lightning.cacheDirectoryPath)
	if err != nil {
		lightning.log.Error("Error connecting to ark: " + err.Error())
		if deactivateErr := lightning.Deactivate(); deactivateErr != nil {
			lightning.log.Error(deactivateErr)
		}
		return err
	}
	lightning.arkClient = &arkClient
	if err = lightning.connect(true); err != nil {
		if deactivateErr := lightning.Deactivate(); deactivateErr != nil {
			lightning.log.Error(deactivateErr)
		}
		return err
	}

	return nil
}

// Connect needs to be called before any requests are made.
func (lightning *Lightning) Connect() {

	// FIXME de-duplicate
	storeSvc, err := store.NewStore(store.Config{
		ConfigStoreType:  arktypes.FileStore,
		AppDataStoreType: arktypes.SQLStore,
		BaseDir:          path.Join(lightning.cacheDirectoryPath, "ark"),
	})
	// if err != nil {
	// 	return fmt.Errorf("failed to setup file store: %s", err)
	// }

	arkClient, err := arksdk.LoadArkClient(storeSvc)
	if err != nil {
		lightning.log.Infof("failed to setup ark client: %s", err)
		return
	}
	lightning.arkClient = &arkClient
	if err := lightning.connect(false); err != nil {
		lightning.log.WithError(err).Warn("LN: Error connecting Ark")
		return
	}

}

// Disconnect closes an active Breez SDK instance. After this, no requests should be made.
func (lightning *Lightning) Disconnect() {
	// if lightning.sdkService != nil {
	// 	if err := lightning.sdkService.Disconnect(); err != nil {
	// 		lightning.log.WithError(err).Warn("BreezSDK: Error disconnecting SDK")
	// 	}

	// 	lightning.sdkService.Destroy()
	// 	lightning.sdkService = nil
	// 	lightning.synced = false
	// }

	if lightning.arkClient != nil {
		arkClient := *lightning.arkClient
		arkClient.Stop()
		if err := arkClient.Lock(context.Background()); err != nil {
			lightning.log.WithError(err).Warn("Error disconnecting Ark")
		}
		lightning.boltzApi = nil
		lightning.swapHandler = nil
	}
}

// Deactivate disconnects the instance, deletes cache folder and changes the config to inactive.
func (lightning *Lightning) Deactivate() error {
	lightningConfig := lightning.backendConfig.LightningConfig()

	if len(lightningConfig.Accounts) == 0 {
		return nil
	}

	// account := lightningConfig.Accounts[0]
	// workingDir := path.Join(lightning.cacheDirectoryPath, accountBreezFolder(account.Code))
	// if err := os.RemoveAll(workingDir); err != nil {
	// 	lightning.log.WithError(err).Error("Error deleting working directory")
	// }

	lightning.Disconnect()
	lightning.arkClient = nil

	if err := os.RemoveAll(path.Join(lightning.cacheDirectoryPath, "ark")); err != nil {
		lightning.log.WithError(err).Error("Error deleting working directory")
	}

	lightningConfig.Accounts = []*config.LightningAccountConfig{}

	if err := lightning.setLightningConfig(lightningConfig); err != nil {
		return err
	}

	return nil
}

// CheckActive returns an error if the lightning service not has been activated.
func (lightning *Lightning) CheckActive() error {
	lightningConfig := lightning.backendConfig.LightningConfig()
	if len(lightningConfig.Accounts) == 0 || lightning.arkClient == nil {
		// if len(lightningConfig.Accounts) == 0 || lightning.sdkService == nil {
		return errp.New("Lightning not initialized")
	}
	return nil
}

// Balance returns the balance of the lightning account.
func (lightning *Lightning) Balance() (*accounts.Balance, error) {
	if err := lightning.CheckActive(); err != nil {
		return nil, err
	}
	arkClient := *lightning.arkClient

	balance, err := arkClient.Balance(context.Background(), false)
	if err != nil {
		lightning.log.Error("Error getting ark balance: " + err.Error())
		return nil, err
	}
	balanceJson, err := json.Marshal(balance)
	if err != nil {
		lightning.log.Error("Error marshalling the ark balance: " + err.Error())
		return nil, err
	}
	lightning.log.Info("Balance: ", string(balanceJson))
	amount := coin.NewAmountFromInt64(int64(balance.OffchainBalance.Total))

	transactions, err := arkClient.GetTransactionHistory(context.Background())
	if err != nil {
		return nil, err
	}
	lightning.log.Info("Transaction history:")
	for _, tx := range transactions {

		transactionJson, err := json.Marshal(tx)
		if err != nil {
			return nil, err
		}
		lightning.log.Info(string(transactionJson))
	}

	spendable, spent, err := arkClient.ListVtxos(context.Background())
	if err != nil {
		return nil, err
	}

	lightning.log.Info("Spendable VTXOs:")
	for _, vtxo := range spendable {
		vtxoJson, err := json.Marshal(vtxo)
		if err != nil {
			return nil, err
		}
		lightning.log.Info(string(vtxoJson))
	}

	lightning.log.Info("Spent VTXOs:")
	for _, vtxo := range spent {
		vtxoJson, err := json.Marshal(vtxo)
		if err != nil {
			return nil, err
		}
		lightning.log.Info(string(vtxoJson))
	}

	return accounts.NewBalance(amount, coin.Amount{}), nil

	// nodeInfo, err := lightning.sdkService.NodeInfo()
	// if err != nil {
	// 	return nil, err
	// }

	// amount := coin.NewAmountFromInt64(int64(nodeInfo.ChannelsBalanceMsat / 1000))
	// return accounts.NewBalance(amount, coin.Amount{}), nil

}

func accountBreezFolder(accountCode types.Code) string {
	return strings.Join([]string{"breez-", string(accountCode)}, "")
}

func (lightning *Lightning) Settle() error {
	if err := lightning.CheckActive(); err != nil {
		return err
	}
	arkClient := *lightning.arkClient
	lightning.log.Info("Going to settle...")
	commitmentTxid, err := arkClient.Settle(context.Background(), arksdk.WithRecoverableVtxos)
	if err != nil {
		return err
	}
	lightning.log.Info("Settlment completed. CommitmentTxId: " + commitmentTxid)
	return nil
}

// connect initializes the connection configuration and calls connect to create a Breez SDK instance.
func (lightning *Lightning) connect(registerNode bool) error {
	lightningConfig := lightning.backendConfig.LightningConfig()
	if len(lightningConfig.Accounts) > 0 && lightning.arkClient != nil {

		arkClient := *lightning.arkClient
		if err := arkClient.Unlock(context.Background(), "password"); err != nil {
			lightning.log.Error("Error unlocking ark client: " + err.Error())
			return err
		}
		lightning.log.Info("Ark connection succeded!!")

		lightning.boltzApi = &boltz.Api{
			// URL:   "https://api.boltz.exchange", // Invoice amount not valid
			// WSURL: "wss://api.boltz.exchange",
			// URL:    "https://boltz.arkade.sh", // Timeout - NPT
			// WSURL:  "wss://boltz.arkade.sh",
			URL:    "https://api.ark.boltz.exchange", // Timeout - NPT
			WSURL:  "wss://api.ark.boltz.exchange",
			Client: *lightning.httpClient,
		}

		cfg, _ := arkClient.GetConfigData(context.Background())
		grpcClient, _ := grpcclient.NewClient(cfg.ServerUrl)
		indexerClient, _ := indexertransport.NewClient(cfg.ServerUrl)
		swapTimeout := uint32(30 * 60) // seconds

		mnemonic := lightning.backendConfig.LightningConfig().Accounts[0].Mnemonic
		lightning.log.Info("Mnemonic: " + mnemonic)
		entropy, err := bip39.EntropyFromMnemonic(mnemonic)
		if err != nil {
			lightning.log.WithError(err).Warn("LN: Error getting entropy")
			return err
		}

		lightning.log.Info("entropy: " + hex.EncodeToString(entropy[:]))
		prvKey, _ := btcec.PrivKeyFromBytes(entropy)

		lightning.log.Info("priv: " + hex.EncodeToString(prvKey.Serialize()))

		lightning.swapHandler = swap.NewSwapHandler(
			arkClient,
			grpcClient,
			indexerClient,
			lightning.boltzApi,
			prvKey.PubKey(),
			swapTimeout,
		)
		lightning.log.Info("Ark address:")

		a, b, c, _ := arkClient.Receive(context.Background())
		lightning.log.Info(a)
		lightning.log.Info(b)
		lightning.log.Info(c)
	}

	// if len(lightningConfig.Accounts) > 0 && lightning.sdkService == nil {
	// 	initializeLogging(lightning.log)

	// 	// At the moment we only support one LN account, but the config files could possibly
	// 	// support multiple accounts, for future extensions.
	// 	account := lightningConfig.Accounts[0]

	// 	seed, err := breez_sdk.MnemonicToSeed(account.Mnemonic)
	// 	if err != nil {
	// 		lightning.log.WithError(err).Error("BreezSDK: MnemonicToSeed failed")
	// 		return err
	// 	}

	// 	var greenlightCredentials *breez_sdk.GreenlightCredentials
	// 	if registerNode {
	// 		_, developerKey, err := util.HTTPGet(lightning.httpClient, greenLightKeyUrl, "", int64(4096))
	// 		if err != nil {
	// 			lightning.log.WithError(err).Error("Greenlight key fetch failed")
	// 			return err
	// 		}

	// 		_, developerCert, err := util.HTTPGet(lightning.httpClient, greenLightCertUrl, "", int64(4096))
	// 		if err != nil {
	// 			lightning.log.WithError(err).Error("Greenlight cert fetch failed")
	// 			return err
	// 		}

	// 		greenlightCredentials = &breez_sdk.GreenlightCredentials{
	// 			DeveloperKey:  developerKey,
	// 			DeveloperCert: developerCert,
	// 		}
	// 	}

	// 	nodeConfig := breez_sdk.NodeConfigGreenlight{
	// 		Config: breez_sdk.GreenlightNodeConfig{
	// 			PartnerCredentials: greenlightCredentials,
	// 			InviteCode:         nil,
	// 		},
	// 	}

	// 	workingDir := path.Join(lightning.cacheDirectoryPath, accountBreezFolder(account.Code))

	// 	if err := os.MkdirAll(workingDir, 0700); err != nil {
	// 		lightning.log.WithError(err).Error("Error creating working directory")
	// 		return err
	// 	}

	// 	breezApiKey, err := lightning.getBreezApiKey()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	config := breez_sdk.DefaultConfig(breez_sdk.EnvironmentTypeProduction, *breezApiKey, nodeConfig)
	// 	config.WorkingDir = workingDir

	// 	connectRequest := breez_sdk.ConnectRequest{
	// 		Config: config,
	// 		Seed:   seed,
	// 	}
	// 	sdkService, err := breez_sdk.Connect(connectRequest, lightning)
	// 	if err != nil {
	// 		lightning.log.WithError(err).Error("BreezSDK: Error connecting SDK")
	// 		return err
	// 	}

	// 	lightning.sdkService = sdkService
	// }
	return nil
}

func (lightning *Lightning) getBreezApiKey() (*string, error) {
	_, breezApiKey, err := util.HTTPGet(lightning.httpClient, breezApiKeyUrl, "", int64(4096))
	if err != nil {
		lightning.log.WithError(err).Error("Breez api key fetch failed")
		return nil, err
	}

	// fetched key could have an unwanted newline, we'll just trim invalid chars for safety.
	trimmedKey := strings.TrimSpace(string(breezApiKey))

	return &trimmedKey, nil
}

func (lightning *Lightning) setLightningConfig(config config.LightningConfig) error {
	if err := lightning.backendConfig.SetLightningConfig(config); err != nil {
		lightning.log.WithError(err).Warn("Error updating lightning config")
		return errp.New("Error updating lightning config")
	}

	lightning.Notify(observable.Event{
		Subject: "lightning/config",
		Action:  action.Reload,
	})

	return nil
}
