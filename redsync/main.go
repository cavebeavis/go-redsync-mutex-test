package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/pkg/errors"
)

var (
	redisAddr = "0.0.0.0:6379"
)

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Network: "tcp",
		// 0.0.0.0 is for standalone aka "go run main.go"
		//Addr:    "0.0.0.0:6379",
		// redis1 is for docker-compose deployment
		Addr: redisAddr,

		// Dialer creates new network connection and has priority over
		// Network and Addr options.
		// Dialer func(ctx context.Context, network, addr string) (net.Conn, error)

		// Hook that is called when new connection is established.
		// OnConnect func(ctx context.Context, cn *Conn) error

		//Username:           "admin",
		//Password:           "password",
		DB:                 0,
		MaxRetries:         3,
		MinRetryBackoff:    time.Millisecond*1,
		MaxRetryBackoff:    time.Millisecond*5,
		DialTimeout:        time.Second*5,
		ReadTimeout:        time.Second*1,
		WriteTimeout:       time.Second*2,
		PoolSize:           2,
		MinIdleConns:       1,
		MaxConnAge:         time.Second*60,
		PoolTimeout:        time.Second*5,
		IdleTimeout:        time.Second*60,
		IdleCheckFrequency: time.Second*5,

		// TLS Config to use. When set TLS will be negotiated.
		// TLSConfig *tls.Config

		// Limiter interface used to implemented circuit breaker or rate limiter.
		// Limiter Limiter
	})

	ctx := context.Background()

	start := time.Now()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Println(errors.Wrap(err, "redis ping"))
		return
	}
	msecs := time.Since(start).Milliseconds()

	log.Print("info", "redis client started", map[string]interface{}{
		"msecs": msecs,
	})


	clientName, _ := redisClient.ClientGetName(ctx).Result()
	clientID, _ := redisClient.ClientID(ctx).Result()
	clientList, _ := redisClient.ClientList(ctx).Result()
	poolStats := fmt.Sprintf("%#v", redisClient.PoolStats())

	log.Print("info", "redis info", map[string]interface{}{
		"clientName": clientName,
		"clientId":   clientID,
		"clientList": clientList,
		"poolStats":  poolStats,
	})

	pool := goredis.NewPool(redisClient)

	// Create an instance of redisync to be used to obtain a mutual exclusion
	// lock.
	rs := redsync.New(pool)

	// Obtain a new mutex by using the same name for all instances wanting the
	// same lock.
	mutexname := "dumb-mutex"
	mutex := rs.NewMutex(mutexname, redsync.WithExpiry(time.Millisecond * 5000))

	rand.Seed(int64(time.Now().Nanosecond()))

	randomFactor := rand.Float64() * 500
	hostname, err := os.Hostname()
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
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond * 57)
		// Obtain a lock for our given mutex. After this is successful, no one else
		// can obtain the same lock (the same mutex name) until we unlock it.
		err := mutex.LockContext(ctx)
		if err != nil {
			rand.Seed(int64(time.Now().Nanosecond()))
			cancel()
			return errors.New("not 10 yet...")
		}

		log.Println("got mutex")
		cancel()
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
		time.Sleep(time.Duration(20) * time.Second)

		cancel()
	}()

	err = backoff.RetryNotify(operation, backoff.WithContext(expBackoff, ctx), notify)
	if err != nil {
		log.Printf("unexpected error: %s", err.Error())
	}
	
	time.Sleep(time.Second * 2)

	log.Println("unlocking mutex")
	ok, err := mutex.Unlock()
	log.Println(ok, err)
}