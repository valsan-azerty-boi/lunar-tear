package service

import (
	"context"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type BannerServiceServer struct {
	pb.UnimplementedBannerServiceServer
	catalog []store.GachaCatalogEntry
}

func NewBannerServiceServer(catalog []store.GachaCatalogEntry) *BannerServiceServer {
	return &BannerServiceServer{catalog: catalog}
}

func (s *BannerServiceServer) GetMamaBanner(ctx context.Context, req *pb.GetMamaBannerRequest) (*pb.GetMamaBannerResponse, error) {
	catalog := s.catalog
	var termLimited []*pb.GachaBanner
	var latestChapter *pb.GachaBanner
	for _, entry := range catalog {
		if entry.GachaLabelType == model.GachaLabelPortalCage || entry.GachaLabelType == model.GachaLabelRecycle {
			continue
		}
		b := &pb.GachaBanner{
			GachaLabelType: entry.GachaLabelType,
			GachaAssetName: entry.BannerAssetName,
			GachaId:        entry.GachaId,
		}
		switch entry.GachaLabelType {
		case model.GachaLabelEvent, model.GachaLabelPremium:
			termLimited = append(termLimited, b)
		case model.GachaLabelChapter:
			if latestChapter == nil || entry.GachaId > latestChapter.GachaId {
				latestChapter = b
			}
		}
	}
	return &pb.GetMamaBannerResponse{
		TermLimitedGacha:   termLimited,
		LatestChapterGacha: latestChapter,
		IsExistUnreadPop:   false,
	}, nil
}
