package main

import (
	"cloud.google.com/go/firestore"
	"context"
)

type Detector struct {
	Name   string `firestore:"name"`
	Status string `firestore:"status"`
}

type Storer interface {
	PutDetector(name, status string) error
	GetDetector(name string) (Detector, error)
}

type storerImpl struct {
	ctx    context.Context
	client *firestore.Client
}

func NewStorer(ctx context.Context, client *firestore.Client) Storer {
	return &storerImpl{
		ctx:    ctx,
		client: client,
	}
}

// PutDetector adds or updates a detector with the given name and id
// to the firestore, returning the key of the newly created entity.
func (s *storerImpl) PutDetector(name, status string) error {
	detector := &Detector{
		Name:   name,
		Status: status,
	}

	_, err := s.client.Collection("detectors").Doc(name).Set(s.ctx, detector, firestore.MergeAll)

	return err
}

func (s *storerImpl) GetDetector(name string) (d Detector, err error) {

	dsnap, err := s.client.Collection("detectors").Doc(name).Get(s.ctx)
	if err != nil {
		return d, err
	}
	dsnap.DataTo(&d)

	return d, nil
}
