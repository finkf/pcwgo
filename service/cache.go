package service

import (
	"errors"

	"github.com/bluele/gcache"
	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db"
	log "github.com/sirupsen/logrus"
)

var (
	projectCache gcache.Cache
	authCache    gcache.Cache
	errNotFound  = errors.New("not found")
)

// Purge purges the project and authentification caches.
func Purge() {
	log.Debugf("cache: purging caches")
	projectCache.Purge()
	authCache.Purge()
}

// RemoveProject remove the given project from the cache.
func RemoveProject(project *db.Project) {
	log.Debugf("removing project id %d from cache", project.ProjectID)
	projectCache.Remove(project.ProjectID)
}

// RemoveSession removes the given session from the cache.
func RemoveSession(session *api.Session) {
	log.Debugf("removing session id %d from cache", session.Auth)
	authCache.Remove(session.Auth)
}

func getCachedProject(id int) (*db.Project, bool, error) {
	p, err := projectCache.Get(id)
	if err != nil && err == errNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	log.Debugf("cache: returning project id: %d", id)
	return p.(*db.Project), true, nil
}

func loadProject(key interface{}) (interface{}, error) {
	id := key.(int)
	log.Debugf("cache: looking up project id: %d", id)
	p, found, err := db.FindProjectByID(pool, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errNotFound
	}
	return p, nil
}

func getCachedSession(id string) (*api.Session, bool, error) {
	s, err := authCache.Get(id)
	if err != nil && err == errNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	log.Debugf("cache: returning session id: %s", id)
	return s.(*api.Session), true, nil
}

func loadSession(key interface{}) (interface{}, error) {
	id := key.(string)
	log.Debugf("cache: looking up session id: %s", id)
	s, found, err := db.FindSessionByID(pool, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errNotFound
	}
	return s, nil
}
