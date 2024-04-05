问题:
设计一个依赖于redis 的分布式锁系统

目标:
1. 易用性
接口简洁，符合直觉，好用
2. 正确性
cover 绝大多数分布式锁使用场景（具体场景在下面逐个分析)
3. 性能
分为锁性能，以及系统性能两块

设计方案:
易用性:
1. 接口简洁:(详见basic_test.go)
// 初始化
redisStore := redis.NewLocalHostRedisStore()
lockManager, err := impl.NewLockManager(redisStore)
// 上锁
lock := lockManager.Lock(ctx, "key") // 或者 lock := lockManager.TryLock(ctx, "key")
// 释放
lock.Release()
// 屏蔽了更复杂的可调控参数，设置为optional
2. 拓展性:
抽象了DB 层，理论上可以支持任何实现相应接口的底层database

正确性场景验证:(详见correctness_test.go)
1. 最基本的,多线程争抢公共资源
详见 TestMultiReadWriteOnCommonVar
2. Blocking (Lock) 和non-blocking（TryLock）API
详见 TestLockAndTryLock
3. KeepAlive 和释放机制
一个任务运行超过Lease时间（少于max lock time），仍能自动续期，但不超过max lock time防止死锁
详见 TestLeaseAndDeadLock
4. 独有锁机制
每个锁分配uuid，防止被其他（已经失效的）获锁错误释放
详见 TestUUIDRelease


性能场景验证:
首先定义性能的标准:
1. 获锁损耗时间
在测试中发现最大的瓶颈在于怎么平衡retry 的时机。尤其是qps内抢占高时，如果retry 的时间点一致，会导致大量资源浪费。
举个例子，如果用普通的exponential backoff 机制:
1ms：假设100个请求同时争夺，此时其中一个获锁，其余99个backoff 1ms
2ms: 99个中1个获锁，剩余98个backoff 2ms
4ms:
...
假如其实每个线程任务处理只需1ms，会导致大部分时间都在等待（而且此时锁为空闲）
所以优化了exponential backoff, 加入了random机制，尽量错开retry时间，使得获锁性能有极大提升

其中retryMinInterval和retryMaxInterval 是两个非常重要的参数，根据不同的场景可以调优（可支持配置）:
TestConcurrentSimpleJob
total time use for 10000 jobs: 1.768995458s
每个线程任务花费极少，几乎都为锁的损耗情况下，能达到1.7s 支持10000个任务上锁以及释放

BenchmarkComplexJob
total time use for 1000 jobs (job time: 5ms): 5.267264416s
每个任务花费5ms 的情况下，1000个任务花费5.2s （损耗仅0.2s/5s = 4%)

2. 平均锁释放
释放基本不存在抢占，只取决于和DB 交互时间，并不是一个瓶颈

3. 系统CPU/mem消耗
对常用对象使用了对象池复用，减少内存毛刺。
此外对于CPU 等待时间，直接利用了Golang 对go routine 调度机制，防止busy waiting 时间
