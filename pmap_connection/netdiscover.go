package main

import (
	"fmt"
	"time"
)

var nodeCount = 50
var linkCount = 8

type node struct {
	outgoing        map[int]bool
	incoming        map[int]bool
	lastConnection  time.Duration
	c               int
	t               int
	confirmedNodes  map[int]bool
	lastMessageSent time.Duration
	connectionSlot  int
}

var nodes = make([]*node, nodeCount)

/*func getTarget(inode, iconn int) int {
	k := 0
	q := []int{0}
	for {
		p := q
		q = []int{}
		// assign one slot for each of the items in p.
		for l := 0; l < linkCount; l++ {
			for _, t := range p {
				k = (k + 1) % nodeCount

				if t == inode && iconn == l {
					return k
				}

				q = append(q, k)
			}
		}
	}
}*/

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
	for t < time.Duration(6000*time.Millisecond) {

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
				// relay confirmed nodes forward, but only if the outgoing node has this node confirmed.
				/*for outgoing := range node.outgoing {
					if !nodes[outgoing].confirmedNodes[iNode] {
						continue
					}
					for confirmedNode := range node.confirmedNodes {
						if !nodes[outgoing].confirmedNodes[confirmedNode] {
							nodes[outgoing].confirmedNodes[confirmedNode] = true
							nodes[outgoing].c++
						}
					}
				}*/

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

				// relay confirmed nodes forward, but only if the outgoing node has this node confirmed.
				/*for outgoing := range node.outgoing {
					if !nodes[outgoing].confirmedNodes[iNode] {
						continue
					}
					for confirmedNode := range node.confirmedNodes {
						if !nodes[outgoing].confirmedNodes[confirmedNode] {
							nodes[outgoing].confirmedNodes[confirmedNode] = true
							nodes[outgoing].c++
						}
					}
				}*/
				node.lastMessageSent = t
			}
		}

		t += 10 * time.Millisecond
	}

	for iNode, node := range nodes {
		outgoing := ""
		for i := range node.outgoing {
			outgoing = outgoing + fmt.Sprintf("%d ", i)
		}
		if len(outgoing) > 0 {
			outgoing = outgoing[:len(outgoing)-1]
		}
		incoming := ""
		for i := range node.incoming {
			incoming = incoming + fmt.Sprintf("%d ", i)
		}
		if len(incoming) > 0 {
			incoming = incoming[:len(incoming)-1]
		}
		hops := getHopDegree(iNode)
		s := fmt.Sprintf("%d: outgoing=[%s] incoming=[%s] forward-visible=%d hops=%d", iNode, outgoing, incoming, len(node.confirmedNodes), hops)
		fmt.Printf("%s\n", s)
	}

}
