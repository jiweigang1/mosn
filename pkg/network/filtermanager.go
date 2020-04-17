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

package network

import (
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)
//管理所有的过滤器
type filterManager struct {
	//负责处理 serverconnection的读取数据？
	upstreamFilters   []*activeReadFilter
	//负责处理下游的请求？
	downstreamFilters []api.WriteFilter
	// ServerConnection 接受到请求的 链接
	conn              api.Connection
	host              api.HostInfo
}
//创建一个 filterManager
func newFilterManager(conn api.Connection) api.FilterManager {
	return &filterManager{
		conn:              conn,
		upstreamFilters:   make([]*activeReadFilter, 0, 8),
		downstreamFilters: make([]api.WriteFilter, 0, 8),
	}
}
//添加一个读的 filter 
func (fm *filterManager) AddReadFilter(rf api.ReadFilter) {
	newArf := &activeReadFilter{
		filter:        rf,
		filterManager: fm,
	}

	rf.InitializeReadFilterCallbacks(newArf)
	fm.upstreamFilters = append(fm.upstreamFilters, newArf)
}

func (fm *filterManager) AddWriteFilter(wf api.WriteFilter) {
	fm.downstreamFilters = append(fm.downstreamFilters, wf)
}

func (fm *filterManager) ListReadFilter() []api.ReadFilter {
	var readFilters []api.ReadFilter

	for _, uf := range fm.upstreamFilters {
		readFilters = append(readFilters, uf.filter)
	}

	return readFilters
}

func (fm *filterManager) ListWriteFilters() []api.WriteFilter {
	return fm.downstreamFilters
}

func (fm *filterManager) InitializeReadFilters() bool {
	if len(fm.upstreamFilters) == 0 {
		return false
	}

	fm.onContinueReading(nil)
	return true
}
// 处理数据的读取
func (fm *filterManager) onContinueReading(filter *activeReadFilter) {
	var index int
	var uf *activeReadFilter

	if filter != nil {
		index = filter.index + 1
	}
	//遍历所有的过滤器进行数据的分发
	for ; index < len(fm.upstreamFilters); index++ {
		uf = fm.upstreamFilters[index]
		uf.index = index

		if !uf.initialized {
			uf.initialized = true

			status := uf.filter.OnNewConnection()

			if status == api.Stop {
				return
			}
		}
		//获取conn 中读取的数据
		buf := fm.conn.GetReadBuffer()

		if buf != nil && buf.Len() > 0 {
			// 执行数据的处理
			status := uf.filter.OnData(buf)

			if status == api.Stop {
				//fm.conn.Write("your data")
				return
			}
		}
	}
}
// 当有数据数据读取的时候，进行调用
func (fm *filterManager) OnRead() {
	fm.onContinueReading(nil)
}

func (fm *filterManager) OnWrite(buf []buffer.IoBuffer) api.FilterStatus {
	for _, df := range fm.downstreamFilters {
		status := df.OnWrite(buf)

		if status == api.Stop {
			return api.Stop
		}
	}

	return api.Continue
}

// as a ReadFilterCallbacks
type activeReadFilter struct {
	index         int
	filter        api.ReadFilter
	filterManager *filterManager
	initialized   bool
}

func (arf *activeReadFilter) Connection() api.Connection {
	return arf.filterManager.conn
}

func (arf *activeReadFilter) ContinueReading() {
	arf.filterManager.onContinueReading(arf)
}

func (arf *activeReadFilter) UpstreamHost() api.HostInfo {
	return arf.filterManager.host
}

func (arf *activeReadFilter) SetUpstreamHost(upstreamHost api.HostInfo) {
	arf.filterManager.host = upstreamHost
}
