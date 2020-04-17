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

package stream

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)
//各种协议解析的工场映射，key 协议的名称，value 协议得解析方式
var streamFactories map[types.ProtocolName]ProtocolStreamFactory

func init() {
	streamFactories = make(map[types.ProtocolName]ProtocolStreamFactory)
}
//注册协议注册的工场
func Register(prot types.ProtocolName, factory ProtocolStreamFactory) {
	streamFactories[prot] = factory
}

func CreateServerStreamConnection(context context.Context, prot api.Protocol, connection api.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {

	if ssc, ok := streamFactories[prot]; ok {
		return ssc.CreateServerStream(context, connection, callbacks)
	}

	return nil
}
//根据协议工场的匹配函数选择协议
func SelectStreamFactoryProtocol(ctx context.Context, prot string, peek []byte) (types.ProtocolName, error) {
	var err error
	var again bool
	//遍历所有的协议解析工厂的匹配函数进行匹配，来选择协议解析工场
	for p, factory := range streamFactories {
		//匹配协议解析函数
		err = factory.ProtocolMatch(ctx, prot, peek)
		//如果返回nil表示匹配成功
		if err == nil {
			//返回匹配的工场
			return p, nil
			//如果返回 eagain，证明读取的字节长度不够，需要重新匹配
		} else if err == EAGAIN {
			again = true
		}
	}
	if again {
		return "", EAGAIN
	} else {
		return "", FAILED
	}
}
