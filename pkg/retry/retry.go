package retry

import (
	retry_v4 "github.com/avast/retry-go/v4"
	log "k8s.io/klog/v2"
)

// DoWithData executes f until it does not return an error
// By default, the number of attempts is 10 with increasing delay between each
func DoWithData(f func() ([]byte, error)) ([]byte, error) {

	return retry_v4.DoWithData(f, retry_v4.OnRetry(func(n uint, err error) {
		log.Errorf("#%d: %s\n", n, err)
	}))

}
