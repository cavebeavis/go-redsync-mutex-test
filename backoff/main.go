package main

import (
	"context"
	"encoding/hex"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
)

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))

	randomFactor := rand.Float64() * 500
	hostname, err := os.Hostname()
	//hostname = "very very very long name of stuff ggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"
	if err == nil && hostname != "" {
		log.Println(hostname)
		hostHex := hex.EncodeToString([]byte(hostname))
		n, err := strconv.ParseInt(string(hostHex), 16, 64)
		if err == nil {
			rand.Seed(n)
			randomFactor = rand.Float64() * 171
		}
		log.Println(n, randomFactor)
	}

	expBackoff := backoff.NewExponentialBackOff()
	
	/*
		ExponentialBackOff is a backoff implementation that increases the backoff period
		for each retry attempt using a randomization function that grows exponentially.

		NextBackOff() is calculated using the following formula:

		randomized interval =
				RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
		
				In other words NextBackOff() will range between the randomization factor 
		percentage below and above the retry interval.

		For example, given the following parameters:

		RetryInterval = 2
		RandomizationFactor = 0.5
		Multiplier = 2

		the actual backoff period used in the next retry attempt will range between 
		1 and 3 seconds, multiplied by the exponential, that is, between 2 and 6 seconds.

		Note: MaxInterval caps the RetryInterval and not the randomized interval.

		If the time elapsed since an ExponentialBackOff instance is created goes past 
		the MaxElapsedTime, then the method NextBackOff() starts returning backoff.Stop.

		The elapsed time can be reset by calling Reset().

		Example: Given the following default arguments, for 10 tries the sequence 
		will be, and assuming we go over the MaxElapsedTime on the 10th try:

		Request #  RetryInterval (seconds)  Randomized Interval (seconds)

		1          0.5                     [0.25,   0.75]
		2          0.75                    [0.375,  1.125]
		3          1.125                   [0.562,  1.687]
		4          1.687                   [0.8435, 2.53]
		5          2.53                    [1.265,  3.795]
		6          3.795                   [1.897,  5.692]
		7          5.692                   [2.846,  8.538]
		8          8.538                   [4.269, 12.807]
		9         12.807                   [6.403, 19.210]
		10        19.210                   backoff.Stop
	*/

	/*
		type ExponentialBackOff struct {
			InitialInterval     time.Duration
			RandomizationFactor float64
			Multiplier          float64
			MaxInterval         time.Duration
			// After MaxElapsedTime the ExponentialBackOff returns Stop.
			// It never stops if MaxElapsedTime == 0.
			MaxElapsedTime time.Duration
			Stop           time.Duration
			Clock          Clock
		}
	*/
	expBackoff.InitialInterval = 7 * time.Millisecond
	expBackoff.RandomizationFactor = rand.Float64()

	rand.Seed(int64(time.Now().Nanosecond()))

	expBackoff.Multiplier = rand.Float64() * randomFactor * 100
	expBackoff.MaxInterval = 6 * time.Second
	expBackoff.MaxElapsedTime = 15 * time.Second

	expBackoff.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retries := 0

	operation := func() error {
		if retries < 10 {
			rand.Seed(int64(time.Now().Nanosecond()))
			return errors.New("not 10 yet...")
		}

		return nil // or an error
	}

	// This function is successful on "successOn" calls.
	notify := func(err error, d time.Duration) {
		if err != nil {
			log.Println(retries, err, d)
			retries++
			return
		}
		log.Println(retries, "success!")
	}

	go func(){
		time.Sleep(time.Duration(rand.Intn(20)) * time.Second)

		cancel()
	}()

	err = backoff.RetryNotify(operation, backoff.WithContext(expBackoff, ctx), notify)
	if err != nil {
		log.Printf("unexpected error: %s", err.Error())
	}
	
}