package main

import (
	"strconv"
	"time"
)

type funcVar func(s string) error

func (f funcVar) Set(s string) error { return f(s) }
func (f funcVar) String() string     { return "" }
func (f funcVar) IsBoolFlag() bool   { return false }

type funcBoolVar func(b bool) error

func (f funcBoolVar) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f funcBoolVar) String() string   { return "" }
func (f funcBoolVar) IsBoolFlag() bool { return true }

type funcDurationVar func(d time.Duration) error

func (f funcDurationVar) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	return f(v)
}
func (f funcDurationVar) String() string   { return "" }
func (f funcDurationVar) IsBoolFlag() bool { return false }

type funcIntVar func(i int) error

func (f funcIntVar) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return err
	}
	return f(int(v))
}
func (f funcIntVar) String() string   { return "" }
func (f funcIntVar) IsBoolFlag() bool { return false }
