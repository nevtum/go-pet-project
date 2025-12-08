package util

import "log"

func MustSucceed(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Must[T any](val T, err error) T {
	MustSucceed(err)
	return val
}
