package main

type Config struct {
	AmountOfWorkers int `json:"amount_of_workers"`
}

type Wallet struct {
	Address 	string `json:"address"`
	PrivateKey 	string `json:"private_key"`
}