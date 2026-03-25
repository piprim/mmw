package config

import "strconv"

type Port int16

func (p Port) String() string {
	return ":" + strconv.Itoa(int(p))
}
