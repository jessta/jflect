// Copyright 2012 The jflect Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
)

// TODO: write proper Usage and README
var (
	fstruct = flag.String("s", "Foo", "struct name for json object")
	debug   = false
)

func main() {
	flag.Parse()
	err := read(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func read(r io.Reader, w io.Writer) error {
	var v interface{}
	err := json.NewDecoder(r).Decode(&v)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	// Open struct
	fmt.Fprintf(buf, "type %s struct {\n", *fstruct)
	b, err := xreflect(v)
	if err != nil {
		return err
	}
	// Write fields to buffer
	buf.Write(b)
	// Close struct
	fmt.Fprintln(buf, "}")
	if debug {
		os.Stdout.WriteString("*********DEBUG***********")
		os.Stdout.Write(buf.Bytes())
		os.Stdout.WriteString("*********DEBUG***********")
	}
	// Pass through gofmt for uniform formatting, and weak syntax check.
	cmd := exec.Command("gofmt")
	cmd.Stdin = buf
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func xreflect(v interface{}) ([]byte, error) {
	var (
		buf = new(bytes.Buffer)
	)
	fields := []Field{}
	switch root := v.(type) {
	case map[string]interface{}:
		for key, val := range root {
			switch j := val.(type) {
			case nil:
				// FIXME: sometimes json service will return nil even though the type is string.
				// go can not convert string to nil and vs versa. Can we assume its a string?
				continue
			case float64:
				fields = append(fields, NewField(key, "int"))
			case map[string]interface{}:
				// If type is map[string]interface{} then we have nested object, Recurse
				fmt.Fprintf(buf, "%s struct {\n", goField(key))
				o, err := xreflect(j)
				if err != nil {
					return nil, err
				}
				_, err = buf.Write(o)
				if err != nil {
					return nil, err
				}
				fmt.Fprintln(buf, "}")
			default:
				fields = append(fields, NewField(key, fmt.Sprintf("%T", val)))
			}
		}
	default:
		return nil, fmt.Errorf("%T: unexpected type", root)
	}
	// Sort and write field buffer last to keep order and formatting.
	sort.Sort(FieldSort(fields))
	for _, f := range fields {
		fmt.Fprintf(buf, "%s %s %s\n", f.name, f.gtype, f.tag)
	}
	return buf.Bytes(), nil
}
