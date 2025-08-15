package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

const defaultTick = 5 * time.Second

func CPUCores() ([]byte, error) {
	cores, err := cpu.Counts(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get cores count: %w", err)
	}

	b, err := IntToBytes(cores)
	if err != nil {
		return nil, fmt.Errorf("failed to convert: %w", err)
	}

	return b, nil
}

func CPUThreads() ([]byte, error) {
	threads, err := cpu.Counts(true)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads count: %w", err)
	}

	b, err := IntToBytes(threads)
	if err != nil {
		return nil, fmt.Errorf("failed to convert: %w", err)
	}

	return b, nil
}

func CPUPercent() ([]byte, error) {
	percent, err := cpu.Percent(defaultTick, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU load: %w", err)
	}

	return Float64ToBytes(percent[0])
}

func CPUPerThread() ([][]byte, error) {
	perThread, err := cpu.Percent(defaultTick, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per thread load: %w", err)
	}

	pt := make([][]byte, len(perThread))

	for i, f := range perThread {
		b, err := Float64ToBytes(f)
		if err != nil {
			return nil, fmt.Errorf("failed to get CPUPerThread: %w", err)
		}

		pt[i] = b
	}

	return pt, nil
}

func CPUByKey(key string) ([][]byte, error) {
	switch key {
	case "cpu.cores":
		b, err := CPUCores()
		if err != nil {
			return nil, err
		}

		bs := make([][]byte, 1)
		bs[0] = b

		return bs, nil
	case "cpu.threads":
		b, err := CPUThreads()
		if err != nil {
			return nil, err
		}

		bs := make([][]byte, 1)
		bs[0] = b

		return bs, nil
	case "cpu.percent":
		b, err := CPUPercent()
		if err != nil {
			return nil, err
		}

		bs := make([][]byte, 1)
		bs[0] = b

		return bs, nil
	case "cpu.percent.thread":
		bs, err := CPUPerThread()
		if err != nil {
			return nil, err
		}

		return bs, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownKeyError, key)
}
