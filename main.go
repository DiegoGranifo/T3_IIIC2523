package main

import (
	"encoding/json"
	"flag"
	"os"
	"log"
	"net"
	"strconv"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Definición de roles
type Role string

const (
	Loyal   Role = "leal"
	Traitor Role = "traidor"
)

// Mensaje que se intercambian los nodos
type Message struct {
	From    int `json:"from"`
	Content int `json:"content"`
}

// Nodo representa un general
type Node struct {
	ID        int
	Role      Role
	Address   string
	Peers     map[int]string
	Listener  net.Listener
	Messages  chan Message
	Decisions map[int]int
	Mutex     sync.Mutex
}

// Nueva instancia de Nodo
func NewNode(id int, role Role, address string, peers map[int]string) *Node {
	return &Node{
		ID:        id,
		Role:      role,
		Address:   address,
		Peers:     peers,
		Messages:  make(chan Message, 100),
		Decisions: make(map[int]int),
	}
}

// Iniciar el servidor del nodo
func (n *Node) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", n.Address)
	if err != nil {
		log.Fatalf("Nodo %d no pudo escuchar en %s: %v", n.ID, n.Address, err)
	}
	n.Listener = listener
	log.Printf("Nodo %d escuchando en %s como %s\n", n.ID, n.Address, n.Role)

	go n.handleMessages()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Nodo %d error al aceptar conexión: %v", n.ID, err)
			continue
		}
		go n.handleConnection(conn)
	}
}

// Manejar conexiones entrantes
func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Nodo %d error al decodificar mensaje: %v", n.ID, err)
		return
	}
	n.Messages <- msg
}

// Procesar mensajes recibidos
func (n *Node) handleMessages() {
	for msg := range n.Messages {
		log.Printf("Nodo %d recibió mensaje de Nodo %d: %d", n.ID, msg.From, msg.Content)
		n.Mutex.Lock()
		n.Decisions[msg.From] = msg.Content
		n.Mutex.Unlock()
	}
}

// Enviar mensaje a un nodo específico
func (n *Node) SendMessage(to int, content int) {
	address, exists := n.Peers[to]
	if !exists {
		log.Printf("Nodo %d no encontró al Nodo %d en peers", n.ID, to)
		return
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Nodo %d no pudo conectar al Nodo %d: %v", n.ID, to, err)
		return
	}
	defer conn.Close()

	msg := Message{
		From:    n.ID,
		Content: content,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(msg); err != nil {
		log.Printf("Nodo %d error al enviar mensaje al Nodo %d: %v", n.ID, to, err)
	}
}

// Broadcast a todos los peers
func (n *Node) Broadcast(content int) {
	for peerID := range n.Peers {
		if n.Role == Traitor && rand.Float32() < 0.5 { // 50% de probabilidad de enviar un contenido aleatorio
			content = rand.Intn(2)
		}
		n.SendMessage(peerID, content)
	}
}

// Ejecutar el Algoritmo del Rey
func (n *Node) RunKingAlgorithm(t int, numGenerals int) {
	rounds := t + 1
	var majorityPlan int
	for round := 1; round <= rounds; round++ {
		log.Printf("Nodo %d inicia la Ronda %d", n.ID, round)

		// Phase 1: Voting Phase
		plan := rand.Intn(2)

		n.Broadcast(plan)
		time.Sleep(5 * time.Second) // Wait to ensure messages are exchanged

		n.Mutex.Lock()
		majorityPlan, votes := n.calculateMajority()
		n.Mutex.Unlock()
		log.Printf("Nodo %d determinó el plan mayoritario '%d' con %d votos", n.ID, majorityPlan, votes)

		// Phase 2: King Phase
		kingID := (round - 1) % numGenerals + 1 // Deterministic King Selection
		if n.ID == kingID {
			n.Broadcast(majorityPlan)
		}
		time.Sleep(5 * time.Second) // Wait for King’s plan

		n.Mutex.Lock()
		kingPlan, exists := n.Decisions[kingID]
		if votes <= numGenerals/2+t && exists {
			log.Printf("Nodo %d cambia su plan al plan del Rey '%d'", n.ID, kingPlan)
			majorityPlan = kingPlan
		}
		n.Mutex.Unlock()

		log.Printf("Nodo %d finalizó la Ronda %d con el plan '%d'", n.ID, round, majorityPlan)
	}

	log.Printf("Nodo %d alcanzó consenso final con plan %d", n.ID,majorityPlan)
	os.Exit(0)
}

// Determinar el plan mayoritario
func (n *Node) calculateMajority() (int, int) {
	voteCount := make(map[int]int)
	for _, plan := range n.Decisions {
		voteCount[plan]++
	}

	majorityPlan := -1
	maxVotes := 0
	for plan, count := range voteCount {
		if count > maxVotes || (count == maxVotes && (majorityPlan == -1 || plan < majorityPlan)) { // Resolve ties deterministically
			majorityPlan = plan
			maxVotes = count
		}
	}
	return majorityPlan, maxVotes
}

// Iniciar el algoritmo
func (n *Node) InitiateAlgorithm() {
	if n.Role == Loyal {
		log.Printf("Nodo %d inicia el algoritmo como leal", n.ID)
	} else {
		log.Printf("Nodo %d inicia el algoritmo como traidor", n.ID)
	}

	// Iniciar el algoritmo del Rey
	numGenerals := len(n.Peers) + 1
	t := (numGenerals - 1) / 4 // Máximo número de traidores permitido
	n.RunKingAlgorithm(t, numGenerals)
}

func main() {
	id := flag.Int("id", 1, "ID del nodo")
	role := flag.String("role", "leal", "Rol del nodo: leal o traidor")
	address := flag.String("address", "0.0.0.0:8000", "Dirección del nodo en formato host:port")
	peersStr := flag.String("peers", "", "Lista de peers en formato id:host:port,id:host:port,...")
	flag.Parse()

	peers := make(map[int]string)
	if *peersStr != "" {
		entries := strings.Split(*peersStr, ",")
		for _, entry := range entries {
			parts := strings.Split(entry, ":")
			if len(parts) != 3 {
				log.Fatalf("Entrada de peers inválida: %s", entry)
			}
			peerID, err := strconv.Atoi(parts[0])
			if err != nil {
				log.Fatalf("ID de peer inválido en entrada: %s", entry)
			}
			peerAddress := parts[1] + ":" + parts[2]
			peers[peerID] = peerAddress
		}
	}

	var nodeRole Role
	if *role == "leal" {
		nodeRole = Loyal
	} else if *role == "traidor" {
		nodeRole = Traitor
	} else {
		log.Fatalf("Rol inválido: %s. Debe ser 'leal' o 'traidor'.", *role)
	}

	node := NewNode(*id, nodeRole, *address, peers)
	var wg sync.WaitGroup
	wg.Add(1)
	go node.Start(&wg)

	time.Sleep(5 * time.Second) // Wait for all nodes to be ready

	node.InitiateAlgorithm()

	wg.Wait()
}
