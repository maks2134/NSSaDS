package network

import (
	"NSSaDS/lab2/internal/domain"
	"fmt"
	"sync"
	"time"
)

type PerformanceMonitor struct {
	mu          sync.RWMutex
	startTime   time.Time
	filename    string
	totalBytes  int64
	transferred int64
	packetsSent uint32
	packetsLost uint32
	retransmits uint32
	bitrates    []float64
	bufferTests map[int]float64
}

func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		bufferTests: make(map[int]float64),
		bitrates:    make([]float64, 0),
	}
}

func (pm *PerformanceMonitor) StartTransfer(filename string, totalBytes int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.filename = filename
	pm.totalBytes = totalBytes
	pm.transferred = 0
	pm.startTime = time.Now()
	pm.packetsSent = 0
	pm.packetsLost = 0
	pm.retransmits = 0
}

func (pm *PerformanceMonitor) UpdateProgress(transferred int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.transferred = transferred

	elapsed := time.Since(pm.startTime).Seconds()
	if elapsed > 0 {
		currentBitrate := float64(transferred) / elapsed / 1024 / 1024
		pm.bitrates = append(pm.bitrates, currentBitrate)

		if len(pm.bitrates) > 100 {
			pm.bitrates = pm.bitrates[1:]
		}
	}
}

func (pm *PerformanceMonitor) GetProgress() *domain.TransferProgress {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	elapsed := time.Since(pm.startTime).Seconds()
	var bitrate float64
	var percentage float64

	if pm.totalBytes > 0 {
		percentage = float64(pm.transferred) / float64(pm.totalBytes) * 100
	}

	if elapsed > 0 {
		bitrate = float64(pm.transferred) / elapsed / 1024 / 1024
	}

	return &domain.TransferProgress{
		FileName:    pm.filename,
		TotalBytes:  pm.totalBytes,
		Transferred: pm.transferred,
		StartTime:   pm.startTime,
		Bitrate:     bitrate,
		Percentage:  percentage,
		PacketsSent: pm.packetsSent,
		PacketsLost: pm.packetsLost,
		Retransmits: pm.retransmits,
	}
}

func (pm *PerformanceMonitor) CalculateOptimalBufferSize() (int, float64) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.bufferTests) == 0 {
		return 8192, 0.0
	}

	var bestSize int
	var bestBitrate float64

	for size, bitrate := range pm.bufferTests {
		if bitrate > bestBitrate {
			bestBitrate = bitrate
			bestSize = size
		}
	}

	return bestSize, bestBitrate
}

func (pm *PerformanceMonitor) CompareWithTCP(tcpBitrate float64) (float64, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.bitrates) == 0 {
		return 0.0, false
	}

	var sum float64
	for _, bitrate := range pm.bitrates {
		sum += bitrate
	}
	avgUDPBitrate := sum / float64(len(pm.bitrates))

	ratio := avgUDPBitrate / tcpBitrate
	isFaster := ratio >= 1.5

	return ratio, isFaster
}

func (pm *PerformanceMonitor) RecordBufferTest(bufferSize int, bitrate float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.bufferTests[bufferSize] = bitrate
}

func (pm *PerformanceMonitor) UpdateStatistics(packetsSent, packetsLost, retransmits uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.packetsSent = packetsSent
	pm.packetsLost = packetsLost
	pm.retransmits = retransmits
}

func (pm *PerformanceMonitor) GetStatistics() (packetsSent, packetsLost, retransmits uint32, avgBitrateValue float64) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.bitrates) > 0 {
		sum := 0.0
		for _, bitrate := range pm.bitrates {
			sum += bitrate
		}
		avgBitrateValue = sum / float64(len(pm.bitrates))
	}

	return pm.packetsSent, pm.packetsLost, pm.retransmits, avgBitrateValue
}

func (pm *PerformanceMonitor) PrintReport() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	fmt.Printf("\n=== Performance Report ===\n")
	fmt.Printf("File: %s\n", pm.filename)
	fmt.Printf("Total Size: %.2f MB\n", float64(pm.totalBytes)/1024/1024)
	fmt.Printf("Transferred: %.2f MB\n", float64(pm.transferred)/1024/1024)
	fmt.Printf("Packets Sent: %d\n", pm.packetsSent)
	fmt.Printf("Packets Lost: %d\n", pm.packetsLost)
	fmt.Printf("Retransmissions: %d\n", pm.retransmits)

	if pm.packetsSent > 0 {
		lossRate := float64(pm.packetsLost) / float64(pm.packetsSent) * 100
		fmt.Printf("Packet Loss Rate: %.2f%%\n", lossRate)
	}

	elapsed := time.Since(pm.startTime).Seconds()
	if elapsed > 0 {
		avgBitrate := float64(pm.transferred) / elapsed / 1024 / 1024
		fmt.Printf("Average Bitrate: %.2f MB/s\n", avgBitrate)
	}

	fmt.Printf("========================\n")
}
