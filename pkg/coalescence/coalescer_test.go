package coalescence

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoalescer(t *testing.T) {
	cacheKey := "cacheKey"
	cacheExpire := 5 * time.Millisecond

	// Should call the operation only once because all the 100 requests came at the same time
	t.Run("ShouldCallOperationOnlyOnce", func(t *testing.T) {
		myCoalescer := NewCoalescer()
		timesInvoked := 0
		operation := func() (interface{}, error) {
			timesInvoked++
			// Simulates some operation. This could be a DB access, HTTP request, wherever...
			time.Sleep(2 * time.Millisecond)
			return timesInvoked, nil
		}

		// Simulates multiple (100) requests for the same operation
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				result, err := myCoalescer.Do(operation, cacheKey, cacheExpire)
				assert.Nil(t, err)
				assert.Equal(t, 1, result)
				wg.Done()
			}()
		}

		wg.Wait()

		// Operations was executed only once for the 100 requests
		assert.Equal(t, 1, timesInvoked)
	})

	// Should still call operation only once because new requests arrived before cache expiration
	t.Run("ShouldStillCallOperationOnlyOnce", func(t *testing.T) {
		myCoalescer := NewCoalescer()
		timesInvoked := 0
		operation := func() (interface{}, error) {
			timesInvoked++
			return timesInvoked, nil
		}

		var wg sync.WaitGroup
		for group := 0; group < 2; group++ {
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					result, err := myCoalescer.Do(operation, cacheKey, cacheExpire)
					assert.Nil(t, err)
					assert.Equal(t, 1, result)
					wg.Done()
				}()
			}

			// Although there is some time between the burst of requests, the operation
			// results will be reused because the cache has not expired.
			time.Sleep(2 * time.Millisecond)
		}

		wg.Wait()
		assert.Equal(t, 1, timesInvoked)
	})

	// Should call operation twice because new requests arrived after cache expiration
	t.Run("ShouldCallOperationTwice", func(t *testing.T) {
		myCoalescer := NewCoalescer()
		timesInvoked := 0
		operation := func() (interface{}, error) {
			timesInvoked++
			return timesInvoked, nil
		}

		var wg sync.WaitGroup
		var expected int = 1
		for group := 0; group < 2; group++ {
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					result, err := myCoalescer.Do(operation, cacheKey, cacheExpire)
					assert.Nil(t, err)
					assert.Equal(t, expected, result)
					wg.Done()
				}()
			}

			// Here the time for the new set of requests is longer than the cache,
			// so the new requests will cause the operation to run again.
			time.Sleep(6 * time.Millisecond)
			expected++
		}

		wg.Wait()
		assert.Equal(t, 2, timesInvoked)
	})
}
