package main

import (
	"strconv"
	"strings"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	if value == "" {
		return nil
	}
	for item := range strings.SplitSeq(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			*s = append(*s, trimmed)
		}
	}
	return nil
}

type optionalInt struct {
	value int
	set   bool
}

func (o *optionalInt) String() string {
	if !o.set {
		return ""
	}
	return strconv.Itoa(o.value)
}

func (o *optionalInt) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

type optionalBool struct {
	value bool
	set   bool
}

func (o *optionalBool) String() string {
	if !o.set {
		return ""
	}
	if o.value {
		return "true"
	}
	return "false"
}

func (o *optionalBool) Set(value string) error {
	if value == "" {
		o.value = true
		o.set = true
		return nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func (o *optionalBool) IsBoolFlag() bool {
	return true
}
