package delayqueue

import (
	"errors"
	"fmt"
	"log"
	"time"
	"hash/fnv"
	"github.com/ouqiang/delay-queue/config"
)

var (
	// 每个定时器对应一个bucket
	timers []*time.Ticker
	// bucket名称chan
	bucketNameChan <-chan string
)

// Init 初始化延时队列
func Init() {
	RedisPool = initRedisPool()
	initTimers()
	bucketNameChan = generateBucketName()
}

// Push 添加一个Job到队列中
func Push(job Job) error {
	if job.Id == "" || job.Topic == "" || job.Delay < 0 || job.TTR <= 0 {
		return errors.New("invalid job")
	}

	err := putJob(job.Id, job)
	if err != nil {
		log.Printf("添加job到job pool失败#job-%+v#%s", job, err.Error())
		return err
	}
	err = pushToBucket(bucket(job.Id), job.Delay, job.Id)
	if err != nil {
		log.Printf("添加job到bucket失败#job-%+v#%s", job, err.Error())
		return err
	}

	return nil
}

// Pop 轮询获取Job
func Pop(topics []string) (*Job, error) {
	jobId, err := blockPopFromReadyQueue(topics, config.Setting.QueueBlockTimeout)
	if err != nil {
		return nil, err
	}

	// 队列为空
	if jobId == "" {
		return nil, nil
	}

	// 获取job元信息
	job, err := getJob(jobId)
	if err != nil {
		return job, err
	}

	// 消息不存在, 可能已被删除
	if job == nil {
		return nil, nil
	}

	timestamp := time.Now().Unix() + job.TTR
	err = pushToBucket(bucket(job.Id), timestamp, job.Id)

	return job, err
}

// Remove 删除Job
func Remove(jobId string) error {
	return removeJob(jobId)
}

// Get 查询Job
func Get(jobId string) (*Job, error) {
	job, err := getJob(jobId)
	if err != nil {
		return job, err
	}

	// 消息不存在, 可能已被删除
	if job == nil {
		return nil, nil
	}
	return job, err
}

func bucket(id string) string {
	if config.Setting.BucketMethod != 0 {
		return <-bucketNameChan
	}
	// 同一 ID 分配相同的 Bucket 名称
	size := config.Setting.BucketSize
	h := hash(id)
	s := fmt.Sprintf(config.Setting.BucketName, int(h) % size + 1)
	return s
}

// string to int32  
func hash(s string) uint32 {
 	h := fnv.New32a()
 	h.Write([]byte(s))
 	return h.Sum32()
}

// 轮询获取bucket名称, 使job分布到不同bucket中, 提高扫描速度
func generateBucketName() <-chan string {
	c := make(chan string)
	go func() {
		i := 1
		for {
			c <- fmt.Sprintf(config.Setting.BucketName, i)
			if i >= config.Setting.BucketSize {
				i = 1
			} else {
				i++
			}
		}
	}()

	return c
}


// 初始化定时器
func initTimers() {
	timers = make([]*time.Ticker, config.Setting.BucketSize)
	var bucketName string
	for i := 0; i < config.Setting.BucketSize; i++ {
		timers[i] = time.NewTicker(1 * time.Second)
		bucketName = fmt.Sprintf(config.Setting.BucketName, i+1)
		go waitTicker(timers[i], bucketName)
	}
}

func waitTicker(timer *time.Ticker, bucketName string) {
	for {
		select {
		case t := <-timer.C:
			tickHandler(t, bucketName)
		}
	}
}

// 扫描bucket, 取出延迟时间小于当前时间的Job
func tickHandler(t time.Time, bucketName string) {
	
	for {
		bucketItem, err := getFromBucket(bucketName)
		if err != nil {
			log.Printf("扫描bucket错误#bucket-%s#%s", bucketName, err.Error())
			return
		}

		// 集合为空
		if bucketItem == nil {
			return
		}

		// 延迟时间未到
		if bucketItem.timestamp > t.Unix() {
			return
		}

		// 延迟时间小于等于当前时间, 取出Job元信息并放入ready queue
		job, err := getJob(bucketItem.jobId)
		if err != nil {
			log.Printf("获取Job元信息失败#bucket-%s#%s", bucketName, err.Error())
			continue
		}

		// job元信息不存在, 从bucket中删除
		if job == nil {
			removeFromBucket(bucketName, bucketItem.jobId)
			continue
		}

		// 再次确认元信息中delay是否小于等于当前时间
		if job.Delay > t.Unix() {
			// 从bucket中删除旧的jobId
			removeFromBucket(bucketName, bucketItem.jobId)
			// 重新计算delay时间并放入bucket中
			pushToBucket(bucketName, job.Delay, bucketItem.jobId)
			continue
		}

		err = pushToReadyQueue(job.Topic, bucketItem.jobId)
		if err != nil {
			log.Printf("JobId放入ready queue失败#bucket-%s#job-%+v#%s",
				bucketName, job, err.Error())
			continue
		}

		// 从bucket中删除
		removeFromBucket(bucketName, bucketItem.jobId)
	}
}
