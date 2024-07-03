package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/schollz/progressbar/v3"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/pbkdf2"

	"github.com/pilanias/go_wallet_genrater/address" // Import the target list
)

const numWorkers = 10

type Result struct {
	Mnemonic string
	Address  string
}

func main() {
	startTime := time.Now()

	// Create a channel to send jobs to workers
	jobs := make(chan int)
	// Create a channel to receive results from workers
	results := make(chan Result)

	var wg sync.WaitGroup
	// Start workers
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	bar := progressbar.NewOptions(-1, // Infinite progress bar
		progressbar.OptionSetDescription("Generating addresses"),
		progressbar.OptionSetWidth(15),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionOnCompletion(func() {
			fmt.Println("\nCompleted!")
		}),
	)

	// Send jobs to workers
	go func() {
		for j := 1; ; j++ {
			jobs <- j
		}
	}()

	found := false
	var foundResult Result

	go func() {
		for result := range results {
			// Check if the address is in the target list
			for _, target := range address.AddressList {
				if strings.Contains(result.Address, target) {
					found = true
					foundResult = result

				}
			}

			if found {
				break

			}

			bar.Add(1)
		}
	}()

	// Close the results channel after all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Wait for results processing to complete
	for range results {
		if found {
			fmt.Printf("\nFound matching address!\nMnemonic: %s\nSolana Address: %s\n\n", foundResult.Mnemonic, foundResult.Address)
			break
		}

	}

	duration := time.Since(startTime)
	seconds := duration.Seconds()

	if found {
		fmt.Printf("\nFound matching address!\nMnemonic: %s\nSolana Address: %s\n\n", foundResult.Mnemonic, foundResult.Address)
		for i := 0; i < 10; i++ {
			fmt.Printf("Mnemonic: %s\nSolana Address: %s\n\n", foundResult.Mnemonic, foundResult.Address)
		}
	} else {
		fmt.Println("No matching address found.")
	}

	fmt.Printf("Elapsed time: %.2f seconds\n", seconds)
}

func worker(id int, jobs <-chan int, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for range jobs {
		// Generate a new random mnemonic phrase
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			log.Fatalf("failed to generate entropy: %v", err)
		}

		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			log.Fatalf("failed to generate mnemonic: %v", err)
		}

		// Step 1: Convert the seed phrase to the master keys
		seed := pbkdf2.Key([]byte(mnemonic), []byte("mnemonic"), 2048, 64, sha512.New)
		h := hmac.New(sha512.New, []byte("ed25519 seed"))
		h.Write(seed)
		I := h.Sum(nil)

		masterPrivateKey := I[:32]
		masterChainCode := I[32:]

		// Step 2: Derive the wallet private key using the path m/44'/501'/0'/0'
		path := []uint32{44 | 0x80000000, 501 | 0x80000000, 0 | 0x80000000, 0 | 0x80000000}
		privateKey, _ := derivePath(masterPrivateKey, masterChainCode, path)

		// Step 3: Generate the Solana key pair
		publicKey := ed25519.NewKeyFromSeed(privateKey)[32:]

		// Step 4: Convert the public key to a Solana address
		address := base58.Encode(publicKey)

		results <- Result{Mnemonic: mnemonic, Address: address}
	}
}

func derivePath(key, chainCode []byte, path []uint32) ([]byte, []byte) {
	for _, index := range path {
		data := make([]byte, 37)
		data[0] = 0
		copy(data[1:33], key)
		data[33] = byte(index >> 24)
		data[34] = byte(index >> 16)
		data[35] = byte(index >> 8)
		data[36] = byte(index)

		h := hmac.New(sha512.New, chainCode)
		h.Write(data)
		I := h.Sum(nil)

		key = I[:32]
		chainCode = I[32:]
	}
	return key, chainCode
}
