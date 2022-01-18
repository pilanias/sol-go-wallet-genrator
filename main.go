package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/rpc"
	"github.com/portto/solana-go-sdk/types"
	"io/ioutil"
	"os"
)

var wallets []Wallet

func LoadConfiguration(file string) (Config, bool) {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		return Config{}, false
	}
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		return Config{}, false
	}

	return config, true
}

func main() {
	config, success := LoadConfiguration("config.json")
	if !success {
		fmt.Println("Failed to load configuration")
		return
	}
	solClient := client.NewClient(rpc.MainnetRPCEndpoint)

	response, err := solClient.GetVersion(context.TODO())
	if err != nil {
		panic(err)
	}
	fmt.Println("SOL Core Version:", response.SolanaCore)

	for i := 0; i < config.AmountOfWorkers; i++ {
		wallet := types.NewAccount()
		encoded := base58.Encode(wallet.PrivateKey)

		wallets = append(wallets, Wallet{
			Address: wallet.PublicKey.ToBase58(),
			PrivateKey: encoded,
		})

		fmt.Println("Generated Wallet:", wallet.PublicKey.ToBase58())
	}

	type Save struct {
		Wallets []Wallet `json:"wallets"`
	}

	jsonWrite, _ := json.Marshal(Save{Wallets: wallets})

	err = ioutil.WriteFile("wallets.json", jsonWrite, 0644)
	if err != nil {
		panic(err)
	}
}