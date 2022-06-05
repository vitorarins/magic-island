package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
)

type Detector struct {
	Name   string `firestore:"name"`
	Status string `firestore:"status"`
}

type Storer interface {
	PutDetector(name, status string) error
	GetDetector(name string) (*Detector, error)
}

type storerImpl struct {
	ctx       context.Context
	client    *firestore.Client
	detectors map[string]*Detector
}

func NewStorer(ctx context.Context, client *firestore.Client) Storer {
	return &storerImpl{
		ctx:       ctx,
		client:    client,
		detectors: make(map[string]*Detector),
	}
}

// PutDetector adds or updates a detector with the given name and id
// to the firestore, returning the key of the newly created entity.
func (s *storerImpl) PutDetector(name, status string) error {
	if name == "" {
		return fmt.Errorf("Name cannot be empty (name: %v, status: %v)", name, status)
	}

	detector := map[string]string{
		"name":   name,
		"status": status,
	}

	_, err := s.client.Collection("detectors").Doc(name).Set(s.ctx, detector, firestore.MergeAll)

	if err == nil {
		s.detectors[name] = &Detector{
			Name:   name,
			Status: status,
		}
	}

	return err
}

func (s *storerImpl) GetDetector(name string) (d *Detector, err error) {
	if name == "" {
		return nil, fmt.Errorf("Cannot get detector with empty name")
	}

	d, ok := s.detectors[name]
	if !ok {
		var detector Detector
		log.Println("Detector not cached, going to firestore...")
		dsnap, err := s.client.Collection("detectors").Doc(name).Get(s.ctx)
		if err != nil {
			return nil, err
		}
		if err := dsnap.DataTo(&detector); err != nil {
			return nil, err
		}
		d = &detector
		s.detectors[name] = d
	}

	return
}
