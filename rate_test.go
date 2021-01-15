package redis_rate_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/time/rate"

	"github.com/molotovtv/redis_rate"
)

func rateLimiter() *redis_rate.Limiter {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{"server0": ":6379"},
	})

	if err := ring.FlushDB(context.Background()).Err(); err != nil {
		panic(err)
	}
	return redis_rate.NewLimiter(ring)
}

func TestAllow(t *testing.T) {
	l := rateLimiter()

	rate, delay, allow := l.Allow(context.Background(), "test_id", 1, time.Minute)
	if !allow {
		t.Fatalf("rate limited with rate %d", rate)
	}
	if rate != 1 {
		t.Fatalf("got %d, wanted 1", rate)
	}
	if delay > time.Minute {
		t.Fatalf("got %s, wanted <= %s", delay, time.Minute)
	}

	rate, _, allow = l.Allow(context.Background(), "test_id", 1, time.Minute)
	if allow {
		t.Fatalf("not rate limited with rate %d", rate)
	}
	if rate != 2 {
		t.Fatalf("got %d, wanted 2", rate)
	}
}

func TestAllowRateMinute(t *testing.T) {
	const n = 2
	const dur = time.Minute

	l := rateLimiter()

	_, allow := l.AllowRate(context.Background(), "rate", n*rate.Every(dur))
	if !allow {
		t.Fatal("rate limited")
	}

	delay, allow := l.AllowRate(context.Background(), "rate", n*rate.Every(dur))
	if allow {
		t.Fatal("not rate limited")
	}
	if !durEqual(delay, dur/n) {
		t.Fatalf("got %s, wanted 0 < dur < %s", delay, dur/n)
	}
}

func TestAllowRateSecond(t *testing.T) {
	const n = 10
	const dur = time.Second

	l := rateLimiter()

	for i := 0; i < n; i++ {
		_, allow := l.AllowRate(context.Background(), "rate", n*rate.Every(dur))
		if !allow {
			t.Fatal("rate limited")
		}
	}

	delay, allow := l.AllowRate(context.Background(), "rate", n*rate.Every(dur))
	if allow {
		t.Fatal("not rate limited")
	}
	if !durEqual(delay, time.Second) {
		t.Fatalf("got %s, wanted 0 < dur < %s", delay, dur)
	}
}

func TestRedisIsDown(t *testing.T) {
	ring := redis.NewRing(&redis.RingOptions{})
	l := redis_rate.NewLimiter(ring)
	l.Fallback = rate.NewLimiter(rate.Every(time.Second), 1)

	rate, _, allow := l.AllowMinute(context.Background(), "test_id", 1)
	if !allow {
		t.Fatalf("rate limited with rate %d", rate)
	}
	if rate != 0 {
		t.Fatalf("got %d, wanted 0", rate)
	}

	rate, _, allow = l.AllowMinute(context.Background(), "test_id", 1)
	if allow {
		t.Fatalf("not rate limited with rate %d", rate)
	}
	if rate != 0 {
		t.Fatalf("got %d, wanted 0", rate)
	}
}

func TestAllowN(t *testing.T) {
	l := rateLimiter()

	rate, delay, allow := l.AllowN(context.Background(), "test_allow_n", 1, time.Minute, 1)
	if !allow {
		t.Fatalf("rate limited with rate %d", rate)
	}
	if rate != 1 {
		t.Fatalf("got %d, wanted 1", rate)
	}
	if delay > time.Minute {
		t.Fatalf("got %s, wanted <= %s", delay, time.Minute)
	}

	l.AllowN(context.Background(), "test_allow_n", 1, time.Minute, 2)

	rate, delay, allow = l.AllowN(context.Background(), "test_allow_n", 1, time.Minute, 0)
	if allow {
		t.Fatalf("should rate limit with rate %d", rate)
	}
	if rate != 3 {
		t.Fatalf("got %d, wanted 3", rate)
	}
	if delay > time.Minute {
		t.Fatalf("got %s, wanted <= %s", delay, time.Minute)
	}
}

func durEqual(got, wanted time.Duration) bool {
	return got > 0 && got < wanted
}
