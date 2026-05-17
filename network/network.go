package network

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"sync"

	"github.com/basnet-tilak/Duniyani/core"
	"github.com/basnet-tilak/Duniyani/database"
)

var (
	CmdVersion = []byte("version")
	CmdTx      = []byte("tx")
	CmdBlock   = []byte("block")
)

// Message is a wire message on the Duniyani network.
type Message struct {
	Command []byte
	Payload []byte
}

// VersionMsg is exchanged during handshake.
type VersionMsg struct {
	Version    uint32
	BestHeight int64
	AddrFrom   string
}

// InvMsg lists block or transaction inventory.
type InvMsg struct {
	Type  string
	Items [][]byte
}

// GetDataMsg requests a specific item.
type GetDataMsg struct {
	Type string
	ID   []byte
}

// TxMsg carries a transaction payload.
type TxMsg struct {
	Tx *core.Transaction
}

// BlockMsg carries a block payload.
type BlockMsg struct {
	Block *core.Block
}

// Peer simulates a network peer connection.
type Peer struct {
	Addr  string
	Inbox chan *Message
}

// NetworkNode simulates a libp2p-style node.
type NetworkNode struct {
	addr     string
	peers    map[string]*Peer
	mu       sync.RWMutex
	incoming chan *Message
	bc       *core.Blockchain
	mempool  *Mempool
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewNetworkNode creates a network node instance.
func NewNetworkNode(addr string, bc *core.Blockchain, mempool *Mempool) *NetworkNode {
	nodeCtx, cancel := context.WithCancel(context.Background())
	return &NetworkNode{
		addr:     addr,
		peers:    make(map[string]*Peer),
		incoming: make(chan *Message, 128),
		bc:       bc,
		mempool:  mempool,
		ctx:      nodeCtx,
		cancel:   cancel,
	}
}

// Start begins processing incoming messages.
func (n *NetworkNode) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	n.cancel = cancel
	n.ctx = ctx
	log.Printf("Network node started at %s", n.addr)

	for {
		select {
		case msg := <-n.incoming:
			if err := n.handleMessage(msg); err != nil {
				log.Printf("network handler error: %v", err)
			}
		case <-ctx.Done():
			log.Printf("network node %s stopped", n.addr)
			return
		}
	}
}

// Stop terminates the network node.
func (n *NetworkNode) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
}

// AddPeer registers a peer for local simulation.
func (n *NetworkNode) AddPeer(addr string, peer *Peer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[addr] = peer
}

// SendToPeer delivers a message to a peer.
func (n *NetworkNode) SendToPeer(addr string, msg *Message) error {
	n.mu.RLock()
	peer, ok := n.peers[addr]
	n.mu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not found", addr)
	}
	select {
	case peer.Inbox <- msg:
		return nil
	default:
		return fmt.Errorf("peer inbox full")
	}
}

// Receive enqueues an inbound message.
func (n *NetworkNode) Receive(msg *Message) {
	select {
	case n.incoming <- msg:
	default:
		log.Println("incoming queue full, dropping message")
	}
}

func (n *NetworkNode) handleMessage(msg *Message) error {
	switch string(msg.Command) {
	case string(CmdVersion):
		var version VersionMsg
		if err := decodePayload(msg.Payload, &version); err != nil {
			return err
		}
		log.Printf("received version from %s height=%d", version.AddrFrom, version.BestHeight)
	case string(CmdTx):
		var txMsg TxMsg
		if err := decodePayload(msg.Payload, &txMsg); err != nil {
			return err
		}
		return n.mempool.AddTransaction(txMsg.Tx)
	case string(CmdBlock):
		var blockMsg BlockMsg
		if err := decodePayload(msg.Payload, &blockMsg); err != nil {
			return err
		}
		return n.bc.AddBlock(blockMsg.Block)
	default:
		log.Printf("unknown command %s", msg.Command)
	}
	return nil
}

func encodePayload(payload interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodePayload(data []byte, payload interface{}) error {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(payload)
}

// NewMessage wraps a command and payload into a Message.
func NewMessage(command []byte, payload interface{}) (*Message, error) {
	encoded, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	return &Message{Command: command, Payload: encoded}, nil
}

type GossipSub struct {
	mu     sync.RWMutex
	topics map[string][]chan []byte
}

func NewGossipSub() *GossipSub {
	return &GossipSub{topics: make(map[string][]chan []byte)}
}

func (gs *GossipSub) Start(ctx context.Context) {
	<-ctx.Done()
}

func (gs *GossipSub) Subscribe(topic string) <-chan []byte {
	ch := make(chan []byte, 10)
	gs.mu.Lock()
	gs.topics[topic] = append(gs.topics[topic], ch)
	gs.mu.Unlock()
	return ch
}

func (gs *GossipSub) Publish(_ context.Context, topic string, msg []byte) error {
	gs.mu.RLock()
	subscribers := append([]chan []byte(nil), gs.topics[topic]...)
	gs.mu.RUnlock()
	for _, ch := range subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
	return nil
}

type MockP2PHost struct {
	addr   string
	pubsub *GossipSub
}

func NewMockP2PHost(addr string, pubsub *GossipSub) *MockP2PHost {
	return &MockP2PHost{addr: addr, pubsub: pubsub}
}

func (h *MockP2PHost) Start(_ context.Context) error {
	return nil
}

func (h *MockP2PHost) Subscribe(topic string) (<-chan []byte, error) {
	return h.pubsub.Subscribe(topic), nil
}

func (h *MockP2PHost) Broadcast(ctx context.Context, topic string, data []byte) error {
	return h.pubsub.Publish(ctx, topic, data)
}

// Mempool represents a thread-safe pool of unconfirmed transactions.
type Mempool struct {
	mu           sync.RWMutex
	transactions map[string]*core.Transaction
	db           *database.Database
}

// NewMempool creates a new mempool.
func NewMempool(db *database.Database) *Mempool {
	return &Mempool{
		transactions: make(map[string]*core.Transaction),
		db:           db,
	}
}

// AddTransaction adds a transaction to the mempool after validation.
func (m *Mempool) AddTransaction(tx *core.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	txID := string(tx.ID)
	if _, ok := m.transactions[txID]; ok {
		return fmt.Errorf("transaction %s already in mempool", txID)
	}

	for _, vin := range tx.Vin {
		key := []byte(fmt.Sprintf("%x:%d", vin.TxID, vin.Vout))
		_, err := m.db.Get(database.ChainStateBucket, key)
		if err == nil {
			return fmt.Errorf("potential double-spend detected for tx %s", txID)
		}
	}

	m.transactions[txID] = tx
	return nil
}

// GetTransactions returns a slice of all transactions in the mempool.
func (m *Mempool) GetTransactions() []*core.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*core.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// RemoveTransaction removes a transaction from the mempool.
func (m *Mempool) RemoveTransaction(txID []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.transactions, string(txID))
}
