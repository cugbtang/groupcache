# groupcache

## Summary

groupcache is a distributed caching and cache-filling library, intended as a
replacement for a pool of memcached nodes in many cases.

For API docs and examples, see http://godoc.org/github.com/golang/groupcache

## Comparison to memcached

### **Like memcached**, groupcache:

 * shards by key to select which peer is responsible for that key

### **Unlike memcached**, groupcache:

 * does not require running a separate set of servers, thus massively
   reducing deployment/configuration pain.  groupcache is a client
   library as well as a server.  It connects to its own peers, forming
   a distributed cache.

 * comes with a cache filling mechanism.  Whereas memcached just says
   "Sorry, cache miss", often resulting in a thundering herd of
   database (or whatever) loads from an unbounded number of clients
   (which has resulted in several fun outages), groupcache coordinates
   cache fills such that only one load in one process of an entire
   replicated set of processes populates the cache, then multiplexes
   the loaded value to all callers.

 * does not support versioned values.  If key "foo" is value "bar",
   key "foo" must always be "bar".  There are neither cache expiration
   times, nor explicit cache evictions.  Thus there is also no CAS,
   nor Increment/Decrement.  This also means that groupcache....

 * ... supports automatic mirroring of super-hot items to multiple
   processes.  This prevents memcached hot spotting where a machine's
   CPU and/or NIC are overloaded by very popular keys/values.

 * is currently only available for Go.  It's very unlikely that I
   (bradfitz@) will port the code to any other language.

## Loading process

In a nutshell, a groupcache lookup of **Get("foo")** looks like:

(On machine #5 of a set of N machines running the same code)

 1. Is the value of "foo" in local memory because it's super hot?  If so, use it.

 2. Is the value of "foo" in local memory because peer #5 (the current
    peer) is the owner of it?  If so, use it.

 3. Amongst all the peers in my set of N, am I the owner of the key
    "foo"?  (e.g. does it consistent hash to 5?)  If so, load it.  If
    other callers come in, via the same process or via RPC requests
    from peers, they block waiting for the load to finish and get the
    same answer.  If not, RPC to the peer that's the owner and get
    the answer.  If the RPC fails, just load it locally (still with
    local dup suppression).

## Users

groupcache is in production use by dl.google.com (its original user),
parts of Blogger, parts of Google Code, parts of Google Fiber, parts
of Google production monitoring systems, etc.

## Presentations

See http://talks.golang.org/2013/oscon-dl.slide

## Help

Use the golang-nuts mailing list for any discussion or questions.

## 产品理念
Groupcache 是一个基于 Go 语言的分布式缓存库，它旨在为大规模的分布式系统提供高效的缓存机制。Groupcache 的设计目标是实现高性能、低延迟和可水平扩展的缓存解决方案。

## 功能介绍

- 分布式缓存管理：Groupcache 允许将缓存数据分布在多个节点上，通过一致性哈希算法来选择缓存节点，实现数据的分片和负载均衡。
- LRU 缓存策略：Groupcache 使用 LRU（最近最少使用）算法来管理缓存项，当缓存空间不足时，会自动淘汰最久未使用的缓存项以腾出空间。
- 并发访问控制：Groupcache 在多个节点之间实现了并发访问控制，通过使用互斥锁和协程来确保并发访问时的数据一致性和线程安全性。
- 自动缓存加载：Groupcache 提供一个 Getter 接口，用户可以实现该接口的 Get 函数来定义如何从后端存储加载数据到缓存中，并且可以通过设置缓存项的过期时间来控制数据的更新。
- 缓存预热：Groupcache 支持在启动时预先加载一部分热门数据到缓存中，以提高缓存的命中率和系统性能。
- 统计信息收集：Groupcache 提供了统计信息，可以跟踪缓存的命中率、获取次数、存储空间使用情况等，方便监控和性能调优。

## 产品设计
- Groupcache 的核心组件是 Group，一个 Group 对应一个缓存实例。
- 每个 Group 包含一个主缓存（MainCache）和一个热点缓存（HotCache）。主缓存用于存储大部分的缓存项，而热点缓存用于存储最近被频繁访问的缓存项。
- Groupcache 使用一致性哈希算法将缓存项映射到多个节点上，每个节点负责管理一部分缓存项。当需要从缓存中获取数据时，Groupcache 会根据 key 使用一致性哈希算法选择对应的节点，并发送请求获取数据。如果节点上没有缓存数据，则会通过 Getter 接口的 Get 函数从后端存储加载数据到缓存中。
- Groupcache 还实现了并发访问控制，通过互斥锁和协程来保证并发访问时的数据一致性和线程安全性。当多个请求同时访问同一个缓存项时，只有第一个请求会从后端加载数据，其他请求会等待并获取已加载的数据。

## 架构设计

GroupCache 的设计思路是将数据分布到多个节点上，并通过缓存和一致性哈希算法，以及避免缓存击穿、重复计算等技术，提高性能和效率。它的核心组件有一致性哈希算法、Getter 接口、多级缓存、分布式缓存、热数据缓存和飞行集群等。以下是 GroupCache 的设计思路和涉及的组件：

- Consistent Hashing：GroupCache 采用了一致性哈希算法，这个算法可用于将键和值映射到服务器集群中的节点。它的优点是在添加或删除节点时，最小化数据迁移。每个节点负责其范围内的键值对，这种范围被称为“虚拟节点”。Consistent Hashing 算法也可以使负载在节点之间均衡分布。

- Getter 接口：GroupCache 可以缓存任何项，它仅执行一个操作：查找并返回给定键的值。当一个 key 没有对应的 value 时，它就会调用一个用户定义的 getter 函数，该函数负责计算这个 key 对应的 value，并将其添加到 GroupCache 中。

- 单机缓存：GroupCache 使用两级缓存，其中第一级是 LRU 缓存（即 mainCache），它存储每个节点负责的本地键值对数据。当需要访问数据时，首先检查此缓存，如果数据存在，则可以直接返回，避免了网络通信开销。

- 分布式缓存：当主缓存未命中时，GroupCache 通过 RPC 调用其他节点来获取数据。GroupCache 将所有节点分成多个基于一致性哈希的分区。它会尝试从负责该 key 的节点中获取值，如果那个节点当前不可用，则会从另一个节点中获取值，并将其添加到本地缓存中。

- 避免缓存击穿：为避免缓存击穿，GroupCache 支持热数据缓存（hotCache），即存储非本地键值对数据。这些数据由于非常热门，频繁被访问，为了避免频繁从其他节点获取，缓存这些数据。通过在本地缓存中维护一些热门数据，可以减少网络传输并提高响应速度。

- 避免重复计算：GroupCache 还支持使用 flightGroup 飞行集群来确保每个键只被获取一次。无论有多少并发调用者，每个键只会被获取一次，无论是从本地还是远程获取，从而避免重复的数据获取操作，提高性能和效率。

## 竞品对比
groupcache 和 memcached 都是常用的缓存解决方案，但它们在实现和功能上存在差异。

- 分布式架构
groupcache 支持分布式缓存，可以在多个节点之间共享缓存数据，并使用 key 分片来选择负责该 key 的节点。而 memcached 只能运行在独立的服务器上，不支持自动缩放和负载均衡。

- 库和服务器
groupcache 是一个客户端库和服务器，不需要单独部署和配置 memcached 服务器。这简化了部署和管理，并且减少了网络延迟和带宽消耗。

- 缓存填充
groupcache 提供了缓存填充机制，当缓存未命中时，只有一个加载过程会填充缓存，然后将加载的值传递给所有调用者。而 memcached 只会在缓存未命中时返回“缓存未命中”错误，从而可能导致大量的数据库或其他数据源加载请求。

- 版本化值和缓存过期
groupcache 不支持带版本的值、缓存过期时间和显式的缓存淘汰。键的值始终保持不变，因此也没有 CAS 或递增/递减操作。相反，memcached 支持缓存过期和显式的淘汰，同时提供 CAS 和递增/递减操作。

综上所述，groupcache 在分布式缓存、缓存填充和服务器库集成等方面比 memcached 更有优势。但是，对于简单的缓存应用，memcached 可能更加轻量级和易于使用。

## 名词解释
- sink，是指一个数据结构或函数，用于承载或接收从其他数据结构或函数中流过来的数据
- AtomicInt类型的目的是提供一种线程安全的方式来处理整数值的原子操作。在多线程并发的环境下，如果多个线程同时对同一个整数值进行修改或读取操作


## 组件/包



## 使用
```go
package main

import (
	"fmt"
	"log"

	"github.com/golang/groupcache"
)

// 用户结构体
type User struct {
	ID   string
	Name string
	Age  int
}

// 获取用户信息的回调函数
func getUserFromDB(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	// 模拟从数据库查询用户信息
	user := User{
		ID:   key,
		Name: "John Doe",
		Age:  30,
	}

	// 将查询结果写入 Sink 对象
	dest.SetProto(&user)
	return nil
}

func main() {
	// 创建 GroupCache 实例
	group := groupcache.NewGroup("userCache", 64<<20, groupcache.GetterFunc(getUserFromDB))

	// 获取用户信息
	var user User
	err := group.Get(nil, "user123", groupcache.ProtoSink(&user))
	if err != nil {
		log.Fatal(err)
	}

	// 打印查询结果
	fmt.Printf("User ID: %s\n", user.ID)
	fmt.Printf("User Name: %s\n", user.Name)
	fmt.Printf("User Age: %d\n", user.Age)
}

```