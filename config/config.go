package config

import (
	"log"
	"os"
	"strconv"
	"gopkg.in/ini.v1"
)

// 解析配置文件

var (
	// Setting 配置实例
	Setting *Config
)

const (
	// DefaultBindAddress 监听地址
	DefaultBindAddress         = "0.0.0.0:9277"
	// DefaultBucketSize bucket数量
	DefaultBucketSize          = 3
	// DefaultBucketName bucket名称
	DefaultBucketName          = "dq_bucket_%d"
	// DefaultQueueName 队列名称
	DefaultQueueName           = "dq_queue_%s"
	// DefaultGetBueketMethod  0 为 hash 算法，1 为轮询算法
	DefaultBueketMethod   = 0
	// DefaultQueueBlockTimeout 轮询队列超时时间
	DefaultQueueBlockTimeout   = 178
	// DefaultRedisHost Redis连接地址
	DefaultRedisHost           = "127.0.0.1:6379"
	// DefaultRedisDb Redis数据库编号
	DefaultRedisDb             = 1
	// DefaultRedisPassword Redis密码
	DefaultRedisPassword       = ""
	// DefaultRedisMaxIdle Redis连接池闲置连接数
	DefaultRedisMaxIdle        = 10
	// DefaultRedisMaxActive Redis连接池最大激活连接数, 0为不限制
	DefaultRedisMaxActive      = 0
	// DefaultRedisConnectTimeout Redis连接超时时间,单位毫秒
	DefaultRedisConnectTimeout = 5000
	// DefaultRedisReadTimeout Redis读取超时时间, 单位毫秒
	DefaultRedisReadTimeout    = 180000
	// DefaultRedisWriteTimeout Redis写入超时时间, 单位毫秒
	DefaultRedisWriteTimeout   = 3000


)

// Config 应用配置
type Config struct {
	BindAddress       string      // http server 监听地址
	BucketSize        int         // bucket数量
	BucketName        string      // bucket在redis中的键名,
	BucketMethod	  int 		  // bucket name 的方法
	QueueName         string      // ready queue在redis中的键名
	QueueBlockTimeout int         // 调用blpop阻塞超时时间, 单位秒, 修改此项, redis.read_timeout必须做相应调整
	Redis             RedisConfig // redis配置
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host           string
	Db             int
	Password       string
	MaxIdle        int // 连接池最大空闲连接数
	MaxActive      int // 连接池最大激活连接数
	ConnectTimeout int // 连接超时, 单位毫秒
	ReadTimeout    int // 读取超时, 单位毫秒
	WriteTimeout   int // 写入超时, 单位毫秒
}

// Init 初始化配置
func Init(path string) {
	Setting = &Config{}
	if path == "" {
		Setting.initDefaultConfig()
		return
	}

	Setting.parse(path)
}

// 解析配置文件
func (config *Config) parse(path string) {
	file, err := ini.Load(path)
	if err != nil {
		log.Fatalf("无法解析配置文件#%s", err.Error())
	}

	section := file.Section("")
	config.BindAddress = GetEnvString("HOST", section.Key("bind_address").MustString(DefaultBindAddress))
	config.BucketSize, _ = GetEnvInt("BUCKET_SIZE" , section.Key("bucket_size").MustInt(DefaultBucketSize))
	config.BucketName = GetEnvString("BUCKET_NAME", section.Key("bucket_name").MustString(DefaultBucketName))
	config.BucketMethod, _ = GetEnvInt("BUCKET_METHOD", section.Key("bucket_method").MustInt(DefaultBueketMethod))
	config.QueueName = GetEnvString("QUEUE_NAME", section.Key("queue_name").MustString(DefaultQueueName))
	config.QueueBlockTimeout = section.Key("queue_block_timeout").MustInt(DefaultQueueBlockTimeout)

	config.Redis.Host = GetEnvString("REDIS_HOST", section.Key("redis.host").MustString(DefaultRedisHost))
	config.Redis.Db, _ = GetEnvInt("REDIS_DB", section.Key("redis.db").MustInt(DefaultRedisDb))
	config.Redis.Password = GetEnvString("REDIS_PASSWORD",section.Key("redis.password").MustString(DefaultRedisPassword))
	config.Redis.MaxIdle, _ = GetEnvInt("REDIS_IDLE", section.Key("redis.max_idle").MustInt(DefaultRedisMaxIdle))
	config.Redis.MaxActive, _ = GetEnvInt("REDIS_ACTIVE", section.Key("redis.max_active").MustInt(DefaultRedisMaxActive))
	config.Redis.ConnectTimeout, _ = GetEnvInt("REDIS_TIMEOUT", section.Key("redis.connect_timeout").MustInt(DefaultRedisConnectTimeout))
	config.Redis.ReadTimeout, _ = GetEnvInt("REDIS_READ_TIMEOUT", section.Key("redis.read_timeout").MustInt(DefaultRedisReadTimeout))
	config.Redis.WriteTimeout, _ = GetEnvInt("REDIS_WRITE_TIMEOUT", section.Key("redis.write_timeout").MustInt(DefaultRedisWriteTimeout))
	
}

// 初始化默认配置
func (config *Config) initDefaultConfig() {
	config.BindAddress = DefaultBindAddress
	config.BucketSize = DefaultBucketSize
	config.BucketName = DefaultBucketName
	config.BucketMethod = DefaultBueketMethod
	config.QueueName = DefaultQueueName
	config.QueueBlockTimeout = DefaultQueueBlockTimeout

	config.Redis.Host = DefaultRedisHost
	config.Redis.Db = DefaultRedisDb
	config.Redis.Password = DefaultRedisPassword
	config.Redis.MaxIdle = DefaultRedisMaxIdle
	config.Redis.MaxActive = DefaultRedisMaxActive
	config.Redis.ConnectTimeout = DefaultRedisConnectTimeout
	config.Redis.ReadTimeout = DefaultRedisReadTimeout
	config.Redis.WriteTimeout = DefaultRedisWriteTimeout
}

// GetEnvString gets the environment variable for a key and if that env-var hasn't been set it returns the default value
func GetEnvString(key string, defaultVal string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		value = defaultVal
	}
	return value
}

// GetEnvBool gets the environment variable for a key and if that env-var hasn't been set it returns the default value
func GetEnvBool(key string, defaultVal bool) (bool, error) {
	envvalue := os.Getenv(key)
	if len(envvalue) == 0 {
		value := defaultVal
		return value, nil
	}

	value, err := strconv.ParseBool(envvalue)
	return value, err
}

// GetEnvInt gets the environment variable for a key and if that env-var hasn't been set it returns the default value. This function is equivalent to ParseInt(s, 10, 0) to convert env-vars to type int
func GetEnvInt(key string, defaultVal int) (int, error) {
	envvalue := os.Getenv(key)
	if len(envvalue) == 0 {
		value := defaultVal
		return value, nil
	}

	value, err := strconv.Atoi(envvalue)
	return value, err
}

// GetEnvFloat gets the environment variable for a key and if that env-var hasn't been set it returns the default value. This function uses bitSize of 64 to convert string to float64.
func GetEnvFloat(key string, defaultVal float64) (float64, error) {
	envvalue := os.Getenv(key)
	if len(envvalue) == 0 {
		value := defaultVal
		return value, nil
	}

	value, err := strconv.ParseFloat(envvalue, 64)
	return value, err
}
