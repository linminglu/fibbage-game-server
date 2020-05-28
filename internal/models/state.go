package models

const (
	WAIT StateType = iota
	ONE
	TWO
	THREE
	FINISH
)

type StateType int
