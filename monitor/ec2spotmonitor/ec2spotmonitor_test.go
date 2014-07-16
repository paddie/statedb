package ec2spotmonitor

import (
	"fmt"
	"github.com/paddie/goamz/ec2"
	"launchpad.net/goamz/aws"
	"testing"
	"time"
)

func TestRealtimeTrace(t *testing.T) {

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
	}
	s := ec2.New(auth, aws.EUWest)

	m, err := NewEC2Monitor(s, "m1.medium",
		"Linux/UNIX",
		"eu-west-1b", nil)

	to := time.Now()
	from := to.AddDate(0, -3, 0)

	trace, err := m.Trace(from, to)
	if err != nil {
		t.Fatal(err)
	}

	if len(trace) == 0 {
		t.Fatal("trace did not produce any history")
	}

	if len(trace) < 5 {
		for i := 0; i < len(trace); i++ {
			pp := trace[i]
			fmt.Printf("\t%.4f - %v\n", pp.Price(), pp.Time())
		}
	} else {
		fmt.Println("last 5 traces:")
		for i := 0; i < 5; i++ {
			pp := trace[i]
			fmt.Printf("\t%.4f - %v\n", pp.Price(), pp.Time())
		}
	}

	pChan, errChan := m.Start(time.Second * 5)

	ticker := time.NewTicker(time.Minute * 5)

	fmt.Println("Starting EC2 Spot Price Monitor.\n\tTimeout: 5 minutes")

	for {
		select {
		case p := <-pChan:
			fmt.Printf("PriceChange: %v On: %v\n", p.Price(), p.Time())
		case err := <-errChan:
			fmt.Println("error: ", err.Error())
			return
		case <-ticker.C:
			fmt.Println("test shut down after 5 minutes")
			return
		}
	}
	fmt.Println("Stopping ticker, and exiting test..")
	ticker.Stop()
	fmt.Println("Done")
}
