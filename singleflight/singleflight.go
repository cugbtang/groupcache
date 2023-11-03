/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package singleflight provides a duplicate function call suppression
// mechanism.
package singleflight

import "sync"

// call 结构体用于记录对某个键值的请求状态信息。
// 它包含了一个等待组 wg、一个结果值 val 和一个错误 err。
// 等待组 wg 用于等待与该键值相关的请求的完成。
// 结果值 val 和错误 err 用于存储请求执行后的结果。
// call is an in-flight or completed Do call
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 结构体代表一个工作组或命名空间，用于执行具有重复抑制功能的工作单元。它包含一个互斥锁 mu 和一个映射 m。
// 互斥锁 mu 用于保护对共享数据的访问，确保只有一个线程可以修改映射 m。
// 映射 m 是一个字符串和 call 结构体的键值对集合，用于存储每个键值的状态信息。
// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// 设计上，flightGroup 的核心思想是利用互斥锁和等待组实现并发控制和结果同步。当调用 Do 方法时，先加锁，然后检查键值是否已存在于映射 m 中。
// 如果存在，表示已经有其他请求在处理相同的键值，当前请求需要等待原始请求完成并返回结果。
// 如果不存在，表示当前请求是第一个请求，需要创建一个新的 call 结构体，并将其添加到映射 m 中。然后释放锁，执行请求的处理函数 fn，并将结果存储在 call 结构体中。
// 请求处理完成后，通过调用等待组的 Done 方法，表示请求已完成。然后再次加锁，从映射 m 中删除该键值对应的 call 结构体，最后释放锁。
// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
