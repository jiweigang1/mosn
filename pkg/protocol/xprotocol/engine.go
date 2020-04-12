/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xprotocol

import (
	"context"
	"errors"

	"mosn.io/mosn/pkg/types"
)

type matchPair struct {
	matchFunc types.ProtocolMatch
	protocol  XProtocol
}

// XEngine is an implementation of the ProtocolEngine interface
type XEngine struct {
	protocols []matchPair
}

// Match use registered matchFunc to recognize corresponding protocol
//执行所有的匹配函数进行匹配
func (engine *XEngine) Match(ctx context.Context, data types.IoBuffer) (types.Protocol, types.MatchResult) {
	//默认无需进行 重新匹配
	again := false
	//遍历所有的注册函数进行匹配
	for idx := range engine.protocols {
		result := engine.protocols[idx].matchFunc(data.Bytes())
		//如果匹配成功，直接返回匹配的协议	
		if result == types.MatchSuccess {
			return engine.protocols[idx].protocol, result
		//如果返回需要重新匹配，暂时进行标示
		} else if result == types.MatchAgain {
			again = true
		}
	}

	// match not success, return failed if all failed; otherwise return again
	// 如果没有匹配成功，但是又重新匹配返回，返回重新匹配状态
	// 需要重新匹配的原因当前读取的字节数量太少，不足于进行匹配
	if again {
		return nil, types.MatchAgain
	//如果不需要重新匹配返回匹配失败	
	} else {
		return nil, types.MatchFailed
	}
}

// Register register protocol, which recognized by the matchFunc
func (engine *XEngine) Register(matchFunc types.ProtocolMatch, protocol types.Protocol) error {
	// check name conflict
	for idx := range engine.protocols {
		if engine.protocols[idx].protocol.Name() == protocol.Name() {
			return errors.New("duplicate protocol register:" + string(protocol.Name()))
		}
	}
	xprotocol, ok := protocol.(XProtocol)
	if !ok {
		return errors.New("protocol is not a instance of XProtocol:" + string(protocol.Name()))
	}

	engine.protocols = append(engine.protocols, matchPair{matchFunc: matchFunc, protocol: xprotocol})
	return nil
}

func NewXEngine(protocols []string) (*XEngine, error) {
	engine := &XEngine{}

	for idx := range protocols {
		name := protocols[idx]

		// get protocol
		protocol := GetProtocol(types.ProtocolName(name))
		if protocol == nil {
			return nil, errors.New("no such protocol:" + name)
		}

		// get matcher
		matchFunc := GetMatcher(types.ProtocolName(name))
		if matchFunc == nil {
			return nil, errors.New("protocol matcher is needed while using multiple protocols:" + name)
		}

		// register
		if err := engine.Register(matchFunc, protocol); err != nil {
			return nil, err
		}
	}
	return engine, nil
}
