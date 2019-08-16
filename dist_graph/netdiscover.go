package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os/exec"
	"sort"
	"time"
)

var nodeCount = 20
var linkCount = 4
var distLinkCount = 2

type node struct {
	outgoing        map[int]bool
	incoming        map[int]bool
	connections     []int
	lastConnection  time.Duration
	c               int
	t               int
	confirmedNodes  map[int]bool
	lastMessageSent time.Duration
	connectionSlot  int
	prev            int
	hops            int
}

var nodes = make([]*node, nodeCount)

func findHostConnectionTarget(inode, iconn, linkCount, nodeCount int) int {
	k := 0
	for n := 0; n <= inode; n++ {
		for l := 0; l < linkCount; l++ {
			k = k + 1
			if l == iconn && inode == n {
				return k % nodeCount
			}
		}
	}
	return -1
}

func getHopDegree(iNode int) int {
	visited := make(map[int]bool)
	q := []int{iNode}
	hops := -1
	for {
		if len(q) == 0 {
			break
		}
		p := q
		q = []int{}
		for _, i := range p {
			for t := range nodes[i].outgoing {
				if !visited[t] {
					q = append(q, t)
					visited[t] = true
				}
			}
			for t := range nodes[i].incoming {
				if !visited[t] {
					q = append(q, t)
					visited[t] = true
				}
			}
		}
		hops++
	}
	return hops
}

func joinLinks(nodeIndex int) []int {
	out := make([]int, len(nodes[nodeIndex].outgoing)+len(nodes[nodeIndex].incoming))
	i := 0
	for t := range nodes[nodeIndex].outgoing {
		out[i] = t
		i++
	}
	for t := range nodes[nodeIndex].incoming {
		out[i] = t
		i++
	}
	return out
}

func route(startNode int, randomSeed int) {
	rnd := rand.New(rand.NewSource(int64(randomSeed)))
	dist := make(map[int]int) // node index -> distance
	prev := make([]int, nodeCount)
	outgoingCount := make([]int, nodeCount)
	infDist := 999999
	q := make([]int, nodeCount)
	for i := 0; i < nodeCount; i++ {
		dist[i] = infDist
		prev[i] = -1
		q[i] = i
	}
	dist[startNode] = 0
	for len(q) > 0 {
		// find the item in q that has the lowest dist.
		smallestDistItemIndex := 0
		for qi, i := range q {
			if dist[i] < dist[smallestDistItemIndex] {
				smallestDistItemIndex = qi
			}
		}
		u := q[smallestDistItemIndex]

		// remove item from q.
		q = append(q[0:smallestDistItemIndex], q[smallestDistItemIndex+1:]...)

		alt := 0
		uConnections := nodes[u].connections
		rnd.Shuffle(len(uConnections), func(i, j int) {
			uConnections[i], uConnections[j] = uConnections[j], uConnections[i]
		})
		// go over the neighbors of u
		for _, v := range uConnections {
			if outgoingCount[u] < distLinkCount {
				alt = dist[u] + 1 // 1 == cost(u, v)
			} else {
				alt = dist[u] + nodeCount
			}
			if alt < dist[v] {
				if prev[v] != -1 {
					outgoingCount[prev[v]]--
				}
				dist[v] = alt
				prev[v] = u
				outgoingCount[u]++
			}
		}
	}

	// copy the result to the nodes.
	for i := 0; i < len(nodes); i++ {
		nodes[i].prev = prev[i]
		nodes[i].hops = dist[i]
	}

}

func main() {
	t := time.Duration(0)
	for i := range nodes {
		nodes[i] = &node{
			outgoing:        make(map[int]bool, 0),
			incoming:        make(map[int]bool, 0),
			confirmedNodes:  make(map[int]bool, 0),
			t:               1,
			lastMessageSent: -time.Second,
		}
	}
	for t < time.Duration(1000*time.Millisecond) {

		// establish connection as needed.
		for iNode, node := range nodes {
			if node.connectionSlot < linkCount && t > node.lastConnection+50*time.Millisecond {
				node.connectionSlot++

				i := findHostConnectionTarget(iNode, node.connectionSlot-1, linkCount, nodeCount)
				if i == iNode || node.outgoing[i] || node.incoming[i] {
					continue
				}
				node.outgoing[i] = true
				nodes[i].incoming[iNode] = true
				node.lastConnection = t
				node.confirmedNodes[i] = true
			}
		}

		// send a message if needed.
		for _, node := range nodes {
			if node.lastMessageSent < t-500*time.Millisecond {
				node.c = 0
				if node.t > 1 {
					node.t = node.t / 2
				}

				// relay confirmed nodes backward.
				for incoming := range node.incoming {
					for confirmedNode := range node.confirmedNodes {
						if !nodes[incoming].confirmedNodes[confirmedNode] && confirmedNode != incoming {
							nodes[incoming].confirmedNodes[confirmedNode] = true
							nodes[incoming].c++
						}
					}
				}

				node.lastMessageSent = t
			} else if node.c > node.t {
				node.c = 0
				node.t = node.t * 2

				// relay confirmed nodes backward.
				for incoming := range node.incoming {
					for confirmedNode := range node.confirmedNodes {
						if !nodes[incoming].confirmedNodes[confirmedNode] && confirmedNode != incoming {
							nodes[incoming].confirmedNodes[confirmedNode] = true
							nodes[incoming].c++
						}
					}
				}

				node.lastMessageSent = t
			}
		}

		t += 10 * time.Millisecond
	}
	startRoute := 7

	// prepare the connections list.
	for iNode, node := range nodes {
		connections := joinLinks(iNode)
		sort.Ints(connections)
		node.connections = connections
	}

	route(startRoute, 0x1234567)

	networkGraph := "digraph G {\n"
	for iNode, node := range nodes {

		for i := range node.outgoing {
			networkGraph += fmt.Sprintf("\tR%d -> R%d\n", iNode, i)
		}
		if node.prev == -1 {
			// it's the originating node.
			networkGraph += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\" color=blue]\n", iNode, iNode, node.hops)
		} else {
			networkGraph += fmt.Sprintf("\tR%d -> R%d [color=red]\n", node.prev, iNode)
			networkGraph += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\"]\n", iNode, iNode, node.hops)
		}
	}
	networkGraph += fmt.Sprintf("}\n")
	ioutil.WriteFile("networkGraph.dot", []byte(networkGraph), 0777)
	cmd := exec.Command("dot", "-Tpng", "networkGraph.dot", "-o", "networkGraph.png")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Unable to run command : %v \n", err)
	}

	networkTree := "digraph G {\n"
	for iNode, node := range nodes {
		if node.prev == -1 {
			// it's the originating node.
			networkTree += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\" color=blue]\n", iNode, iNode, node.hops)
		} else {
			networkTree += fmt.Sprintf("\tR%d -> R%d [color=red]\n", node.prev, iNode)
			networkTree += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\"]\n", iNode, iNode, node.hops)
		}
	}
	networkTree += fmt.Sprintf("}\n")
	ioutil.WriteFile("networkTree1.dot", []byte(networkTree), 0777)
	cmd = exec.Command("dot", "-Tpng", "networkTree1.dot", "-o", "networkTree1.png")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Unable to run command : %v \n", err)
	}

	// reset the connections list.
	for _, node := range nodes {
		sort.Ints(node.connections)
	}

	route(startRoute, 0x493274982)

	networkTree = "digraph G {\n"
	for iNode, node := range nodes {
		if node.prev == -1 {
			// it's the originating node.
			networkTree += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\" color=blue]\n", iNode, iNode, node.hops)
		} else {
			networkTree += fmt.Sprintf("\tR%d -> R%d [color=red]\n", node.prev, iNode)
			networkTree += fmt.Sprintf("\tR%d [label=\"R%d\nhops=%d\"]\n", iNode, iNode, node.hops)
		}
	}
	networkTree += fmt.Sprintf("}\n")
	ioutil.WriteFile("networkTree2.dot", []byte(networkTree), 0777)
	cmd = exec.Command("dot", "-Tpng", "networkTree2.dot", "-o", "networkTree2.png")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Unable to run command : %v \n", err)
	}
}
