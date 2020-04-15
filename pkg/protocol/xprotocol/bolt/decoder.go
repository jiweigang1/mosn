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
//负责协议解码，解码包含请求解码和响应解码
import (
	"context"
	"encoding/binary"
	"strconv"

	"mosn.io/mosn/pkg/variable"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)
//解码请求信息
func decodeRequest(ctx context.Context, data types.IoBuffer, oneway bool) (cmd interface{}, err error) {
	//获取请求的长度
	bytesLen := data.Len()
	//获取请求的字节信息
	bytes := data.Bytes()

	//如果字节的内容不是完整内容，返回 1. least bytes to decode header is RequestHeaderLen(22)
	if bytesLen < RequestHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(bytes[14:16])
	headerLen := binary.BigEndian.Uint16(bytes[16:18])
	//获取 content 的长度
	contentLen := binary.BigEndian.Uint32(bytes[18:22])
	// 一个桢的总长度
	frameLen := RequestHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	// 如果小于一个桢的长度，直接返回，等待下次的读取
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	buf := bufferByContext(ctx)
	request := &buf.request
	//默认设置为请求
	cmdType := CmdTypeRequest
	//如果是单项的请求，设置为单项请求方式
	if oneway {
		cmdType = CmdTypeRequestOneway
	}
	// 解码请求的 header
	request.RequestHeader = RequestHeader{
		Protocol:   ProtocolCode,
		CmdType:    cmdType,
		CmdCode:    binary.BigEndian.Uint16(bytes[2:4]),
		Version:    bytes[4],
		RequestId:  binary.BigEndian.Uint32(bytes[5:9]),
		Codec:      bytes[9],
		Timeout:    int32(binary.BigEndian.Uint32(bytes[10:14])),
		ClassLen:   classLen,
		HeaderLen:  headerLen,
		ContentLen: contentLen,
	}
	request.Data = buffer.GetIoBuffer(frameLen)

	// 4. set timeout to notify proxy
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, strconv.Itoa(int(request.Timeout)))

	//5. copy data for io multiplexing
	request.Data.Write(bytes[:frameLen])
	request.rawData = request.Data.Bytes()

	//6. process wrappers: Class, Header, Content, Data
	headerIndex := RequestHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	request.rawMeta = request.rawData[:RequestHeaderLen]
	if classLen > 0 {
		request.rawClass = request.rawData[RequestHeaderLen:headerIndex]
		request.Class = string(request.rawClass)
	}
	if headerLen > 0 {
		request.rawHeader = request.rawData[headerIndex:contentIndex]
		err = xprotocol.DecodeHeader(request.rawHeader, &request.Header)
	}
	// 如果 content 的长度大于零 需要解析 请求 body的内容
	if contentLen > 0 {
		request.rawContent = request.rawData[contentIndex:]
		request.Content = buffer.NewIoBufferBytes(request.rawContent)
	}
	return request, err
}
//解码响应信息
func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is ResponseHeaderLen(20)
	if bytesLen < ResponseHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(bytes[12:14])
	headerLen := binary.BigEndian.Uint16(bytes[14:16])
	contentLen := binary.BigEndian.Uint32(bytes[16:20])

	frameLen := ResponseHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	buf := bufferByContext(ctx)
	response := &buf.response

	response.ResponseHeader = ResponseHeader{
		Protocol:       ProtocolCode,
		CmdType:        CmdTypeResponse,
		CmdCode:        binary.BigEndian.Uint16(bytes[2:4]),
		Version:        bytes[4],
		RequestId:      binary.BigEndian.Uint32(bytes[5:9]),
		Codec:          bytes[9],
		ResponseStatus: binary.BigEndian.Uint16(bytes[10:12]),
		ClassLen:       classLen,
		HeaderLen:      headerLen,
		ContentLen:     contentLen,
	}
	response.Data = buffer.GetIoBuffer(frameLen)

	//TODO: test recycle by model, so we can recycle request/response models, headers also
	//4. copy data for io multiplexing
	response.Data.Write(bytes[:frameLen])
	response.rawData = response.Data.Bytes()

	//5. process wrappers: Class, Header, Content, Data
	headerIndex := ResponseHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	response.rawMeta = response.rawData[:ResponseHeaderLen]
	if classLen > 0 {
		response.rawClass = response.rawData[ResponseHeaderLen:headerIndex]
		response.Class = string(response.rawClass)
	}
	if headerLen > 0 {
		response.rawHeader = response.rawData[headerIndex:contentIndex]
		err = xprotocol.DecodeHeader(response.rawHeader, &response.Header)
	}
	if contentLen > 0 {
		response.rawContent = response.rawData[contentIndex:]
		response.Content = buffer.NewIoBufferBytes(response.rawContent)
	}
	return response, err
}
