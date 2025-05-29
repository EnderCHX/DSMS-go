package auth

import (
	"sync"
	"testing"
)

func TestLogin(t *testing.T) {
	errCount := 0
	wg := &sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			_, _, err := Login("test", "test")
			if err != nil {
				errCount++
				//t.Log(err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	t.Log("login error count:", errCount)
}
