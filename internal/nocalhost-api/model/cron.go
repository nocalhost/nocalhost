package model

type Job struct {
	Spec func() string
	Exec func()
}
