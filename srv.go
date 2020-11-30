package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type cfg struct {
	rate     int
	Interval time.Duration
}

type server struct {
	cfg        cfg
	listener   net.Listener
	grpcServer *grpc.Server
}

func (s *server) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Capabilities not implemented")
}

func (s *server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}

func (s *server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Set not implemented")
}

func (s *server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("failed subscribe server rcv: %v", err)
			return err
		}
		peer, _ := peer.FromContext(stream.Context())
		log.Printf("rcv subscribeRequest from peer=%s, req=%+v", peer.Addr, req)

		switch req := req.Request.(type) {
		case *gnmi.SubscribeRequest_Subscribe:
			log.Printf("starting send goroutine for subscribe request: %+v", req)
			go s.sendSubResponse(stream, s.cfg.rate, s.cfg.Interval)
		}
	}
}

func (s *server) sendSubResponse(ss gnmi.GNMI_SubscribeServer, rate int, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		for i := 0; i < rate; i++ {
			now := time.Now().UnixNano()
			subResp := &gnmi.SubscribeResponse{
				Response: &gnmi.SubscribeResponse_Update{
					Update: &gnmi.Notification{
						Timestamp: now,
						Prefix: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "state"},
								{Name: "port", Key: map[string]string{
									"port-id": fmt.Sprintf("1/1/%d", i+1),
								}},
								{Name: "ethernet"},
								{Name: "statistics"},
							},
						},
						Update: []*gnmi.Update{
							{
								Path: &gnmi.Path{
									Elem: []*gnmi.PathElem{
										{Name: "in-octets"},
									},
								},
								Val: &gnmi.TypedValue{
									Value: &gnmi.TypedValue_JsonVal{
										JsonVal: []byte(strconv.Itoa(int(now))),
									},
								},
							},
						},
					},
				},
			}
			err := ss.Send(subResp)
			if err != nil {
				log.Printf("worker-%d failed to send response: %v", i, err)
				return
			}
		}
	}
}
