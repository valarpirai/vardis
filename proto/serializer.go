package proto

// Implemention of RESP (REdis Serialization Protocol)
// RESP - Serializer and Deserializer
// https://redis.io/topics/protocol

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Data types and constants
const (
	typeSimpleStrings = "+"
	typeErrors        = "-"
	typeIntegers      = ":"
	typeBulkStrings   = "$"
	typeArrays        = "*"

	typeNull            = "-1"
	crlf                = "\r\n"
	bulkStringMaxLength = 512 * 1024 * 1024
)

type Request struct {
	rawCmd string
	cmd    string
	key    string
	args   []string
	err    bool
}

// ********* Request Interface Start ************

type RequestInterface interface {
	String() string
	Command() string
	Key() string
	Value() string
	ArgsLength() int
	Args() []string
	Error() bool
	CommandLength() int
}

func (req *Request) SetRawCommand(cmd string) {
	req.rawCmd = cmd
}

func (req *Request) String() string {
	if req.Error() {
		return ""
	}
	return req.rawCmd
}

func (req *Request) Command() string {
	if req.Error() {
		return ""
	}
	return req.cmd
}

func (req *Request) Key() string {
	if req.Error() {
		return ""
	}
	return req.key
}

func (req *Request) Value() string {
	if req.Error() || req.ArgsLength() < 1 {
		return ""
	}
	return req.args[0]
}

func (req *Request) CommandLength() int {
	if req.Error() {
		return 0
	}
	// TODO - handle if key is nil
	return 2 + req.ArgsLength()
}

func (req *Request) ArgsLength() int {
	if req.Error() || nil == req.Args() {
		return 0
	}
	return len(req.args)
}

func (req *Request) Args() []string {
	if req.Error() {
		return nil
	}
	return req.args
}

func (req *Request) Error() bool {
	return req.err
}

// ********* Request Interface End ************

// EncodeString encodes a simple string
func EncodeString(s string) []byte {
	if strings.ContainsAny(s, crlf) {
		panic("SimpleString cannot contain a CR or LF character")
	}
	return []byte(typeSimpleStrings + s + crlf)
}

// EncodeError encodes a error string
func EncodeError(s string) []byte {
	return []byte(typeErrors + s + crlf)
}

// EncodeInt encodes an int
func EncodeInt(s int64) []byte {
	return []byte(typeIntegers + strconv.FormatInt(s, 10) + crlf)
}

// EncodeBulkString encodes a bulk string
func EncodeBulkString(s string) []byte {
	if len(s) > bulkStringMaxLength {
		panic("BulkString is over 512 MB")
	}
	return []byte(typeBulkStrings + strconv.Itoa(len(s)) + crlf + s + crlf)
}

// EncodeNull encodes null value
func EncodeNull() []byte {
	return []byte(typeBulkStrings + typeNull + crlf)
}

// EncodeNullArray encodes null array
func EncodeNullArray() []byte {
	return []byte(typeArrays + typeNull + crlf)
}

// EncodeArray encode a slice of byte slice. It accepts the results of other encode method including itself.
// For example: EncodeArray([][]byte{EncodeInt(1), EncodeNull()})
func EncodeArray(s [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(typeArrays)
	buf.WriteString(strconv.Itoa(len(s)))
	buf.WriteString(crlf)
	for _, val := range s {
		buf.Write(val)
	}
	return buf.Bytes()
}

// Decode decode from reader
func Decode(reader *bufio.Reader) (result interface{}, rawCmd string, err error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	rawCmd = line
	lineLen := len(line)
	if lineLen < 3 {
		err = fmt.Errorf(`line is too short: %#v`, line)
		return
	}
	if line[lineLen-2] != '\r' || line[lineLen-1] != '\n' {
		err = fmt.Errorf("invalid CRLF: %#v", line)
		return
	}

	msgType, line := string(line[0]), line[1:lineLen-2]
	switch msgType {
	case typeSimpleStrings:
		result = line
	case typeErrors:
		result = errors.New(line)
	case typeIntegers:
		result, err = strconv.ParseInt(line, 10, 64)
	case typeBulkStrings:
		var length int

		if length, err = strconv.Atoi(line); err != nil || length == -1 {
			return
		}

		if length > bulkStringMaxLength || length < -1 {
			err = fmt.Errorf("invalid Bulk Strings length: %#v", length)
			return
		}
		buff := make([]byte, length+2)
		_, err = io.ReadFull(reader, buff)
		if err != nil {
			return
		}
		if buff[length] != '\r' || buff[length+1] != '\n' {
			err = fmt.Errorf("invalid CRLF: %#v", string(buff))
			return
		}
		result = string(buff[:length])
	case typeArrays:
		var length int
		length, err = strconv.Atoi(line)
		if length == -1 {
			return
		}
		array := make([]interface{}, length)
		for i := 0; i < length; i++ {
			var raw string
			array[i], raw, err = Decode(reader)
			if err != nil {
				return
			}
			rawCmd += raw
		}
		result = array
	default:
		// Default treat as String
		// log.Debugln(command)
		result = msgType + line
	}
	return
}

func ParseCommand(cmd interface{}) *Request {
	var request = new(Request)
	var args []string
	switch cmd.(type) {
	case []interface{}:
		// Parsing Array of commands
		args = ConvertInterfaceArrToStringArr(cmd)
	case interface{}:
		args = strings.Split(cmd.(string), " ")
	default:
		request.err = true
	}

	if request.Error() == false {
		argsLen := len(args)
		if argsLen > 2 {
			request.cmd = args[0]
			request.key = args[1]
			request.args = args[2:argsLen]
		} else if argsLen > 1 {
			request.cmd = args[0]
			request.key = args[1]
		} else {
			request.cmd = args[0]
		}
		request.cmd = strings.ToUpper(request.cmd)
	}
	return request
}

func ConvertInterfaceArrToStringArr(netData interface{}) (aString []string) {
	params, ok := netData.([]interface{})
	if !ok {
		return
	}
	aString = make([]string, len(params))
	for i, v := range params {
		aString[i] = v.(string)
	}
	return
}
