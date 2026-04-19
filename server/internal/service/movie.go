package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

type MovieServiceServer struct {
	pb.UnimplementedMovieServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewMovieServiceServer(users store.UserRepository, sessions store.SessionRepository) *MovieServiceServer {
	return &MovieServiceServer{users: users, sessions: sessions}
}

func (s *MovieServiceServer) SaveViewedMovie(ctx context.Context, req *pb.SaveViewedMovieRequest) (*pb.SaveViewedMovieResponse, error) {
	log.Printf("[MovieService] SaveViewedMovie: movieIds=%v", req.MovieId)

	userId := currentUserId(ctx, s.users, s.sessions)
	now := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, mid := range req.MovieId {
			user.ViewedMovies[mid] = now
		}
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, []string{"IUserMovie"}))

	return &pb.SaveViewedMovieResponse{
		DiffUserData: diff,
	}, nil
}
