package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"time"

	"golang.org/x/crypto/sha3"
)

var verbose = false
var delay = false
var channelCount = 1
var risk = 60
var v float64

type Data struct {
	Raw       string
	Encrypted string
}

type Response struct {
	Encrypted string
	ID        int64
	OK        bool
}
type Worker interface {
	Feed(d Data)
	Start()
	Kill()
	GetResponseChannel() *chan Response
}
type worker struct {
	ID       int64
	Response chan Response
	Source   chan Data
	Quit     chan struct{}
}

func NewWorker(ID int64, respChannel chan Response, quitChan chan struct{}) Worker {
	return &worker{
		ID:       ID,
		Source:   make(chan Data),
		Response: respChannel,
		Quit:     quitChan,
	}
}
func (w *worker) GetResponseChannel() *chan Response {
	return &w.Response
}
func (w *worker) Feed(d Data) {
	w.Source <- d
}
func (w *worker) Kill() {

}
func (w *worker) Start() {
	go func() {
		if verbose {
			fmt.Println("worker", w.ID, "started")
		}
		select {
		case msg := <-w.Source:
			res := Response{
				Encrypted: msg.Encrypted,
				ID:        w.ID,
				OK:        w.Compare(msg),
			}
			w.Response <- res
			if verbose {
				fmt.Println("worker", w.ID, "voted for ", res.OK)
			}

		case <-w.Quit:
			fmt.Println("1")

			return
		}
	}()
}

func (w *worker) Compare(d Data) bool {
	if delay {
		//sleep for random time
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	}
	h := sha3.New256()

	if w.ID > int64(v+math.Copysign(0.01, v)) {
		d.Raw = "bad data"
	}
	return string(h.Sum([]byte(d.Raw))) == d.Encrypted
}

func main() {
	flag.IntVar(&channelCount, "c", 1, "number of channels")
	flag.IntVar(&risk, "r", 60, "risk chance percent")
	flag.BoolVar(&delay, "d", false, "delay")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.Parse()
	v = ((float64(risk) * float64(channelCount)) / float64(100))
	//create workers
	var channelsCount = channelCount
	var workers = make([]Worker, channelsCount)
	var responseChannels = make(chan Response, channelsCount)
	var globalQuit = make(chan struct{})

	//==

	h := sha3.New256()
	dummyString := "Hello World"
	var d Data = Data{Raw: dummyString, Encrypted: string(h.Sum([]byte(dummyString)))}

	//==

	for id := range workers {
		workers[id] = NewWorker(int64(id), responseChannels, globalQuit)
		go workers[id].Start()

	}

	//feed workers

	for id := range workers {
		workers[id].Feed(d)
	}

	//get responses

	votes := getResp(responseChannels, channelsCount, globalQuit)
	fmt.Println("election result:", calculateVotes(votes))
}
func calculateVotes(votes map[int64]bool) bool {
	var votesCount = 0
	for _, v := range votes {
		if v {
			votesCount++
		}
	}

	fmt.Printf("we got %s%d%s votes ;%s %.2f%% %s of voters votes true\n",
		string([]byte{27, 91, 51, 53, 109}),
		len(votes),
		string([]byte{27, 91, 48, 109}),
		string([]byte{27, 91, 57, 55, 59, 52, 50, 109}),
		float64(votesCount)/float64(len(votes))*100,
		string([]byte{27, 91, 48, 109}))
	return votesCount >= (len(votes)/2)+1
}
func getResp(responseChannels chan Response, channelsCount int, globalQuit chan struct{}) map[int64]bool {
	respCount := 0
	votes := make(map[int64]bool, channelsCount)
	for {
		select {
		case resp := <-responseChannels:
			votes[resp.ID] = resp.OK
			respCount++
			if respCount == channelsCount {
				close(globalQuit)
				close(responseChannels)
				return votes
			}
		}
	}
}
