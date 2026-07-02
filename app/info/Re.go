package info

import ()

// controller ajax返回
type Re struct {
	Ok   bool
	Code int
	Msg  string
	Id   string
	Url  string
	List interface{}
	Item interface{}
}

func NewRe() Re {
	return Re{Ok: false}
}
