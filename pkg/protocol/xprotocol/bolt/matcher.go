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

package bolt

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)
//注册协议的类型
func init() {
	//注册协议的类型，名称和匹配的函数
	xprotocol.RegisterMatcher(ProtocolName, boltMatcher)
} 
// predicate first byte '0x1'

//根据预先读取的字节进行匹配
func boltMatcher(data []byte) types.MatchResult {
	//获取读取字节的长度
	length := len(data)
	//如果长度为0 证明还没有对到有效数据，需要进行重新匹配
	if length == 0 {
		return types.MatchAgain
	}
	//bolt 协议通过第一个字节就可以判断出来，如果匹配返回匹配成功
	if data[0] == ProtocolCode {
		return types.MatchSuccess
	}
	//返回匹配失败
	return types.MatchFailed
}
