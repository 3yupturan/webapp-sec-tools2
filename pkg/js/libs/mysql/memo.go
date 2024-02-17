package mysql

import (
	"errors"
	"fmt"

	"github.com/projectdiscovery/nuclei/v3/pkg/protocols/common/protocolstate"
)

func memoizedconnectWithDSN(dsn string) (bool, error) {
	hash := "connectWithDSN:" + dsn

	v, err, _ := protocolstate.Memoizer.Do(hash, func() (interface{}, error) {
		return connectWithDSN(dsn)
	})
	if err != nil {
		return false, err
	}
	if value, ok := v.(bool); ok {
		return value, nil
	}

	return false, errors.New("could not convert cached result")
}

func memoizedisMySQL(host string, port int) (bool, error) {
	hash := "isMySQL:" + host + ":" + fmt.Sprint(port)

	v, err, _ := protocolstate.Memoizer.Do(hash, func() (interface{}, error) {
		return isMySQL(host, port)
	})
	if err != nil {
		return false, err
	}
	if value, ok := v.(bool); ok {
		return value, nil
	}

	return false, errors.New("could not convert cached result")
}

func memoizedfingerprintMySQL(host string, port int) (MySQLInfo, error) {
	hash := "fingerprintMySQL:" + host + ":" + fmt.Sprint(port)

	v, err, _ := protocolstate.Memoizer.Do(hash, func() (interface{}, error) {
		return fingerprintMySQL(host, port)
	})
	if err != nil {
		return MySQLInfo{}, err
	}
	if value, ok := v.(MySQLInfo); ok {
		return value, nil
	}

	return MySQLInfo{}, errors.New("could not convert cached result")
}
