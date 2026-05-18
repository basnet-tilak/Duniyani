package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/basnet-tilak/Duniyani/consensus"
	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/database"
	"github.com/basnet-tilak/Duniyani/economics"
	"github.com/basnet-tilak/Duniyani/network"
	"github.com/basnet-tilak/Duniyani/wallet"
)

const (
	dbPath       = "./duniyani_db"
	defaultPeers = "127.0.0.1:9001"
)

// Node orchestrates the Duniyani runtime.
type Node struct {
	bc        *core.Blockchain
	utxoSet   *core.UTXOSet
	wallet    *wallet.Wallet
	net       *network.NetworkNode
	consensus consensus.ConsensusEngine
	mempool   *network.Mempool
	db        *database.Database
}

func main() {
	runNode := flag.Bool("node", false, "Start the Duniyani node services")
	minerAddress := flag.String("miner", "", "Enable mining and send rewards to this address")
	walletOnly := flag.Bool("wallet", false, "Generate a wallet address and exit")
	flag.Parse()

	if *walletOnly {
		w := wallet.NewWallet()
		fmt.Printf("Duniyani wallet address: %s\n", w.GetAddress())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, closeDB := database.InitDatabase(dbPath)
	defer closeDB()

	w := wallet.NewWallet()
	log.Printf("Wallet initialized. Address: %s", w.GetAddress())

	bc, err := core.LoadBlockchain(db)
	if err != nil {
		blockchainErr := err
		if errors.Is(err, database.ErrNotFound) {
			log.Println("No existing blockchain found. Creating genesis block...")
			genesis := economics.CreateGenesisBlock(w.GetAddress())
			bc, err = core.CreateBlockchain(db, genesis)
			if err != nil {
				log.Panicf("Failed to create genesis blockchain: %v", err)
			}
		} else {
			log.Panicf("Failed to load blockchain: %v", blockchainErr)
		}
	}

	utxoSet := core.NewUTXOSet(db)
	if err := utxoSet.Reindex(); err != nil {
		log.Panicf("Failed to reindex UTXO set: %v", err)
	}

	mempool := network.NewMempool(db)
	net := network.NewNetworkNode(defaultPeers, bc, mempool)

	// Initialize PoUW Consensus with a mock Enclave ML-DSA Public Key (empty for local dev)
	engine := consensus.NewPoUWEngine(20, []byte{})

	node := &Node{
		bc:        bc,
		utxoSet:   utxoSet,
		wallet:    w,
		net:       net,
		consensus: engine,
		mempool:   mempool,
		db:        db,
	}

	if *runNode {
		log.Println("Starting Duniyani node services...")
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := node.net.ListenQUIC(); err != nil {
				log.Fatalf("QUIC listener failed: %v", err)
			}
			node.net.Start(ctx)
		}()

		if *minerAddress != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				node.startMining(ctx, *minerAddress)
			}()
		}

		<-ctx.Done()
		stop()
		wg.Wait()
		log.Println("Duniyani node stopped gracefully.")
		return
	}

	// If not running as a node, print wallet info and exit.
	fmt.Println("Duniyani node started in passive mode. Use --node to activate full node services.")
}

func (n *Node) startMining(ctx context.Context, minerAddress string) {
	log.Println("Mining loop started.")

	for {
		select {
		case <-ctx.Done():
			log.Println("Mining loop received a shutdown signal.")
			return
		default:
			lastBlockHash := n.bc.GetLastBlockHash()
			coinbaseTx := economics.NewCoinbaseTx(minerAddress, "", n.bc.Height()+1)

			// Retrieve pending transactions from the mempool
			mempoolTxs := n.mempool.GetTransactions()
			txs := make([]*core.Transaction, 0, len(mempoolTxs)+1)
			txs = append(txs, coinbaseTx)
			txs = append(txs, mempoolTxs...)

			newBlock := core.NewBlock(txs, lastBlockHash, 1, n.consensus.DifficultyTarget())

			if err := n.consensus.Mine(newBlock); err != nil {
				log.Printf("Mining failed: %v", err)
				continue
			}

			if err := n.bc.AddBlock(newBlock); err != nil {
				log.Printf("Failed to add a new block: %v", err)
				continue
			}

			// Broadcast the newly mined block to the network
			n.net.BroadcastBlock(newBlock)

			// Evict included transactions from the mempool
			for _, tx := range mempoolTxs {
				n.mempool.RemoveTransaction(tx.ID)
			}

			log.Printf("Mined new block: %x", newBlock.Header.Hash())
			time.Sleep(5 * time.Second)
		}
	}
}
