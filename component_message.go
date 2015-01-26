package casper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gogap/casper/utils"
	uuid "github.com/nu7hatch/gouuid"
)

type componentCommands map[string]interface{}
type componentContext map[string]interface{}

type callChain struct {
	ComponentName string
	Endpoint      string
}

type ComponentMetadata struct {
	Name   string `json:"name"`
	MQType string `json:"mq_type"`
	In     string `json:"in"`
}

type ComponentMessage struct {
	Id       string               `json:"id"`
	entrance *ComponentMetadata   `json:"entrance"`
	graph    []*ComponentMetadata `json:"graph"`
	chain    []string             `json:"chain"`
	Payload  *Payload             `json:"payload"`
}

type Payload struct {
	Code    uint64            `json:"code"`
	Message string            `json:"message"`
	context componentContext  `json:"context"`
	command componentCommands `json:"command"`
	result  []byte            `json:"result,omitempty"`
}

func NewComponentMessage(entrance *ComponentMetadata, result interface{}) (msg *ComponentMessage, err error) {
	msgId := ""
	if u, e := uuid.NewV4(); e != nil {
		err = e
		return
	} else {
		msgId = u.String()
	}

	msg = &ComponentMessage{
		Id:       msgId,
		entrance: entrance,
		graph:    nil,
		chain:    nil,
		Payload: &Payload{
			Code:    0,
			Message: "OK",
			context: make(map[string]interface{}),
			command: make(map[string]interface{}),
			result:  nil}}

	return msg, msg.Payload.setResult(result)

}

func (p *ComponentMessage) SetEntrance(entrance ComponentMetadata) {
	p.entrance = &entrance
}

func (p *ComponentMessage) TopGraph() *ComponentMetadata {
	if len(p.graph) >= 1 {
		return p.graph[0]
	}

	return nil
}

func (p *ComponentMessage) PopGraph() *ComponentMetadata {
	if len(p.graph) >= 1 {
		p.graph = p.graph[1:]
	}
	if len(p.graph) >= 1 {
		return p.graph[0]
	}

	return nil
}

func (p *ComponentMessage) Serialize() ([]byte, error) {
	type Msg struct {
		Id       string               `json:"id"`
		Entrance *ComponentMetadata   `json:"entrance"`
		Graph    []*ComponentMetadata `json:"graph"`
		Chain    []string             `json:"chain"`
		Payload  struct {
			Code    uint64            `json:"code"`
			Message string            `json:"message"`
			Context componentContext  `json:"context"`
			Command componentCommands `json:"command"`
			Result  []byte            `json:"result"`
		} `json:"payload"`
	}

	tmp := &Msg{}
	tmp.Id = p.Id
	tmp.Entrance = p.entrance
	tmp.Graph = p.graph
	tmp.Chain = p.chain
	if p.Payload != nil {
		tmp.Payload.Code = p.Payload.Code
		tmp.Payload.Message = p.Payload.Message
		tmp.Payload.Context = p.Payload.context
		tmp.Payload.Command = p.Payload.command
		tmp.Payload.Result = p.Payload.result
	}

	return json.Marshal(tmp)
}

func (p *ComponentMessage) FromJson(jsonStr []byte) (err error) {
	var tmp struct {
		Id       string               `json:"id"`
		Entrance *ComponentMetadata   `json:"entrance"`
		Graph    []*ComponentMetadata `json:"graph"`
		Chain    []string             `json:"chain"`
		Payload  struct {
			Code    uint64            `json:"code"`
			Message string            `json:"message"`
			Context componentContext  `json:"context,omitempty"`
			Command componentCommands `json:"command,omitempty"`
			Result  []byte            `json:"result"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(jsonStr, &tmp); err != nil {
		return err
	}

	p.Id = tmp.Id
	p.entrance = tmp.Entrance
	p.graph = tmp.Graph
	p.chain = tmp.Chain
	p.Payload = &Payload{
		Code:    tmp.Payload.Code,
		Message: tmp.Payload.Message,
		context: tmp.Payload.Context,
		command: tmp.Payload.Command,
		result:  tmp.Payload.Result}

	return nil
}

func (p *Payload) UnmarshalResult(v interface{}) error {
	if p.result == nil {
		return nil
	}

	if p.result[0] == '"' {
		p.result = p.result[1:(len(p.result) - 1)]
	}

	dst, err := base64.StdEncoding.DecodeString(string(p.result))
	if err != nil {
		return json.Unmarshal(p.result, v)
	} else {
		return json.Unmarshal(dst, v)
	}
}

func (p *Payload) setResult(v interface{}) (err error) {
	if v == nil {
		p.result = nil
		return nil
	}

	if !utils.IsStruct(v) && !utils.IsStructArray(v) {
		err = fmt.Errorf("worker return must struct or []struct")
		return
	}

	p.result, err = json.Marshal(v)

	return
}

func (p *Payload) GetResult() []byte {
	return p.result
}

func (p *Payload) SetContext(key string, val interface{}) {
	if p.context == nil {
		p.context = make(map[string]interface{})
	}
	p.context[key] = val
}

func (p *Payload) GetContext(key string) (val interface{}, exist bool) {
	if p.context == nil {
		return nil, false
	}
	val, exist = p.context[key]
	return
}

func (p *Payload) GetContextString(key string) (val string, err error) {
	if p.context == nil {
		return "", fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if strVal, ok := v.(string); ok {
		val = strVal
		return
	}

	return
}

func (p *Payload) GetContextStringArray(key string) (vals []string, err error) {
	if p.context == nil {
		return nil, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if vInterfaces, ok := v.([]interface{}); ok {
		tmpArray := []string{}
		for i, vStr := range vInterfaces {
			if str, ok := vStr.(string); ok {
				tmpArray = append(tmpArray, str)
			} else {
				err = fmt.Errorf("the context key of %s's value type at index of %d is not string", key, i)
				return
			}
		}
		vals = tmpArray
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not array", key)
		return
	}
	return
}

func (p *Payload) GetContextInt(key string) (val int, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int", key)
	}
	return
}

func (p *Payload) GetContextInt32(key string) (val int32, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int32); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int32", key)
	}
	return
}

func (p *Payload) GetContextInt64(key string) (val int64, err error) {
	if p.context == nil {
		return 0, fmt.Errorf("the context container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int64); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of context key %s is not int64", key)
	}
	return
}

func (p *Payload) GetContextObject(key string, v interface{}) (err error) {
	if v == nil {
		err = fmt.Errorf("the v should not be nil, it should be a Pointer")
		return
	}

	if p.context == nil {
		return fmt.Errorf("the context container is nil")
	}

	if val, exist := p.context[key]; !exist {
		err = fmt.Errorf("the context key of %s is not exist", key)
		return
	} else if val == nil {
		err = fmt.Errorf("the context key of %s is exist, but the value is nil", key)
		return
	} else {
		if bJson, e := json.Marshal(val); e != nil {
			err = fmt.Errorf("marshal object of %s to json failed, error is:%v", key, e)
			return
		} else if e := json.Unmarshal(bJson, v); e != nil {
			err = fmt.Errorf("unmarshal json to object %s failed, error is:%v", key, e)
			return
		}
	}
	return
}

func (p *Payload) SetCommand(key string, command interface{}) {
	if p.command == nil {
		p.command = make(map[string]interface{})
	}
	p.command[key] = command
}

func (p *Payload) AppendCommand(key string, command interface{}) {
	if p.command == nil {
		p.command = make(map[string]interface{})
	}

	if tmp, ok := p.command[key]; ok {
		if reflect.TypeOf(tmp) == reflect.TypeOf(command) {
			switch reflect.TypeOf(command).Kind() {
			case reflect.Map:
				for k, v := range command.(map[string]interface{}) {
					p.command[key].(map[string]interface{})[k] = v
				}
			case reflect.Slice:
				for i := 0; i < len(command.([]interface{})); i++ {
					p.command[key] = append(p.command[key].([]interface{}), command.([]interface{})[i])
				}
			}
		}
	} else {
		p.command[key] = command
	}
}

func (p *Payload) GetCommand(key string) (val interface{}, exist bool) {
	if p.command == nil {
		return nil, false
	}
	val, exist = p.command[key]
	return
}

func (p *Payload) GetCommandString(key string) (val string, err error) {
	if p.command == nil {
		return "", fmt.Errorf("the command container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	}

	if strVal, ok := v.(string); ok {
		val = strVal
		return
	}

	return
}

func (p *Payload) GetCommandStringArray(key string) (vals []string, err error) {
	if p.command == nil {
		return nil, fmt.Errorf("the command container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	}

	if vInterfaces, ok := v.([]interface{}); ok {
		tmpArray := []string{}
		for i, vStr := range vInterfaces {
			if str, ok := vStr.(string); ok {
				tmpArray = append(tmpArray, str)
			} else {
				err = fmt.Errorf("the command key of %s's value type at index of %d is not string", key, i)
				return
			}
		}
		vals = tmpArray
		return
	} else {
		err = fmt.Errorf("the type of command key %s is not array", key)
		return
	}
	return
}

func (p *Payload) GetCommandInt(key string) (val int, err error) {
	if p.command == nil {
		return 0, fmt.Errorf("the command container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of command key %s is not int", key)
	}
	return
}

func (p *Payload) GetCommandInt32(key string) (val int32, err error) {
	if p.command == nil {
		return 0, fmt.Errorf("the command container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int32); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of command key %s is not int32", key)
	}
	return
}

func (p *Payload) GetCommandInt64(key string) (val int64, err error) {
	if p.command == nil {
		return 0, fmt.Errorf("the command container is nil")
	}

	var v interface{}
	exist := false
	if v, exist = p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	}

	if intVal, ok := v.(int64); ok {
		val = intVal
		return
	} else {
		err = fmt.Errorf("the type of command key %s is not int64", key)
	}
	return
}

func (p *Payload) GetCommandObject(key string, v interface{}) (err error) {
	if v == nil {
		err = fmt.Errorf("the v should not be nil, it should be a Pointer")
		return
	}

	if p.command == nil {
		return fmt.Errorf("the command container is nil")
	}

	if val, exist := p.command[key]; !exist {
		err = fmt.Errorf("the command key of %s is not exist", key)
		return
	} else if val == nil {
		err = fmt.Errorf("the command key of %s is exist, but the value is nil", key)
		return
	} else {
		var bJson []byte
		var e error
		if bJson, e = json.Marshal(val); e != nil {
			err = fmt.Errorf("marshal object of %s to json failed, error is:%v", key, e)
			return
		}

		if e := json.Unmarshal(bJson, v); e != nil {
			err = fmt.Errorf("unmarshal json to object %s failed, error is:%v", key, e)
			return
		}
	}
	return
}
