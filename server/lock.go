package main

import "sync"

var Locks = make(map[string]*sync.Mutex)
var LocksLock sync.Mutex

func Lock(key string) {
	LocksLock.Lock()

	// get the lock by key; create it if it doesn't exist
	if _, ok := Locks[key]; !ok {
		Locks[key] = new(sync.Mutex)
	}
	lock := Locks[key]

	// unlock the lock lock before we wait on the specific lock
	LocksLock.Unlock()

	lock.Lock()
}

func Unlock(key string) {
	LocksLock.Lock()

	// get the lock by key
	if _, ok := Locks[key]; !ok {
		LocksLock.Unlock()
		panic("no such lock: " + key)
	}
	lock := Locks[key]

	// unlock the lock lock before we wait on the specific lock
	LocksLock.Unlock()

	lock.Unlock()
}
