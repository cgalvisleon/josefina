package node

import (
	"time"

	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

var cache *catalog.Model

/**
* initCache: Initializes the cache model
* @return error
**/
func initCache() error {
	if cache != nil {
		return nil
	}

	db, err := node.coreDb()
	if err != nil {
		return err
	}

	cache, err = db.NewModel("", "cache", true, 1)
	if err != nil {
		return err
	}
	if err := cache.Init(); err != nil {
		return err
	}

	return nil
}

/**
* setCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func setCache(key string, value interface{}, duration time.Duration) (*mem.Entry, error) {
	result, err := mem.Set(key, value, duration)
	if err != nil {
		return nil, err
	}

	err = initCache()
	if err != nil {
		return nil, err
	}

	if duration == 0 {
		err := cache.Put(key, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* SetCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func (s *Node) SetCache(key string, value interface{}, duration time.Duration) error {
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.SetCache(key, value, time.Time{}, duration)
	}

	now := timezone.Now()
	err := s.follow.SetCache(key, value, time.Time{}, duration)
	if err != nil {
		return err
	}

	go func() {
		for _, peer := range node.Peers {
			if peer.Addr == s.Address() {
				continue
			}

			if peer.Addr == leader.Addr {
				res := node.Request(leader, "Leader.SetCache", key, value, now, duration)
				if res.Error != nil {
					return
				}
			} else {
				res := node.Request(peer, "Follow.SetCache", key, value, now, duration)
				if res.Error != nil {
					return
				}
			}
		}
	}()

	return nil
}

/**
* DeleteCache: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func (s *Node) DeleteCache(key string) error {
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.DeleteCache(key)
	}

	err := s.follow.DeleteCache(key)
	if err != nil {
		return err
	}

	go func() {
		for _, peer := range node.Peers {
			if peer.Addr == s.Address() {
				continue
			}

			if peer.Addr == leader.Addr {
				res := node.Request(leader, "Leader.DeleteCache", key)
				if res.Error != nil {
					return
				}
			} else {
				res := node.Request(peer, "Follow.DeleteCache", key)
				if res.Error != nil {
					return
				}
			}
		}
	}()

	return nil
}

/**
* ExistsCache: Checks if a cache value exists
* @param key string
* @return bool
**/
func (s *Node) ExistsCache(key string) (bool, error) {
	_, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.ExistsCache(key)
	}

	return s.follow.ExistsCache(key)
}

/**
* GetCache: Gets a cache value
* @param key string
* @return *mem.Entry
**/
func (s *Node) GetCache(key string) (*mem.Entry, bool) {
	value, exists := mem.GetEntry(key)
	if exists {
		return value, true
	}

	set := func(result *mem.Entry, exists bool) (*mem.Entry, bool) {
		expiration := result.Expiration
		if expiration != 0 {
			expiration = result.Expiration - time.Since(result.LastUpdate)
		}
		mem.Set(key, result.Value, expiration)
		return result, exists
	}

	err := initCache()
	if err != nil {
		return nil, false
	}

	result := mem.Entry{}
	exists, err = cache.Get(key, &result)
	if err != nil {
		return nil, false
	}

	if !exists {
		return nil, false
	}

	return set(&result, exists)
}
