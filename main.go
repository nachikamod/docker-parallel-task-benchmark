package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

type Job struct {
	Id  int
	Cli *client.Client
}

type Result struct {
	Id     int     `json:"id"`
	Result int     `json:"result"`
	Time   float64 `json:"time"`
}

const (
	CPU_COUNT      = 2
	POOL_SIZE      = 8
	CPU_PERCENTAGE = 50
	JOB_SIZE       = 100
	BASE_IMG       = "gcc:11.2.1_git20220219-r2"
	WORKDIR        = "/home"
)

func RemoveContainer(cli *client.Client, ctx context.Context, id string) error {
	return cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func worker(id int, jobs <-chan Job, results chan<- Result) {
	for j := range jobs {
		start := time.Now()
		ctx := context.Background()

		cont, err := j.Cli.ContainerCreate(ctx, &container.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Image:        BASE_IMG,
			WorkingDir:   WORKDIR,
			OpenStdin:    true,
		}, &container.HostConfig{
			Resources: container.Resources{
				CPUCount:   CPU_COUNT,
				CPUPercent: CPU_PERCENTAGE,
			},
		}, nil, nil, fmt.Sprintf("gcc-worker-%d", j.Id))

		if err != nil {
			log.Println(err)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		tar, err := archive.TarWithOptions(`D:\Workspace\Overcompute\Benchmark\Performance\Dense\test`, &archive.TarOptions{})
		if err != nil {
			log.Println(err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		err = j.Cli.CopyToContainer(ctx, cont.ID, WORKDIR, tar, types.CopyToContainerOptions{})
		if err != nil {
			log.Println(err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		err = j.Cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{})
		if err != nil {
			log.Println(err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		execID, err := j.Cli.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
			AttachStderr: true,
			AttachStdout: true,
			Tty:          true,
			WorkingDir:   WORKDIR,
			Cmd:          []string{"sh", "-c", "g++ test.cpp && ./a.out"},
		})

		if err != nil {
			log.Println("Error executing :", err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		resp, err := j.Cli.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
		if err != nil {
			log.Println("Error reading :", err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		var outBuf, errBuf bytes.Buffer
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
		if err != nil {
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		stdout, err := ioutil.ReadAll(&outBuf)
		if err != nil {
			log.Println(err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		stderr, err := ioutil.ReadAll(&errBuf)
		if err != nil {
			log.Println(err)
			RemoveContainer(j.Cli, ctx, cont.ID)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		fmt.Println("Stdout :", string(stdout), "Stderr :", string(stderr))

		err = RemoveContainer(j.Cli, ctx, cont.ID)

		if err != nil {
			log.Println(err)
			elapsed := time.Since(start)
			results <- Result{
				Id:     j.Id,
				Result: 0,
				Time:   elapsed.Seconds(),
			}
			return
		}

		elapsed := time.Since(start)
		results <- Result{
			Id:     j.Id,
			Result: 1,
			Time:   elapsed.Seconds(),
		}
	}
}

func main() {

	start := time.Now()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	jobs := make(chan Job, JOB_SIZE)
	results := make(chan Result, JOB_SIZE)

	for w := 1; w <= POOL_SIZE; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= JOB_SIZE; j++ {
		jobs <- Job{
			Id:  j,
			Cli: cli,
		}
	}
	close(jobs)

	records := [][]string{
		{"ID", "Time"},
	}

	for a := 1; a <= JOB_SIZE; a++ {

		result := <-results

		records = append(records, []string{
			fmt.Sprint(result.Id),
			fmt.Sprint(result.Time),
		})

		resultJSON, _ := json.MarshalIndent(result, "", "\t")

		fmt.Println("Results :", string(resultJSON))
	}

	elapsed := time.Since(start)

	log.Printf("Jobs executed in : %f", elapsed.Seconds())

	f, err := os.Create(fmt.Sprintf(`D:\Workspace\Overcompute\Benchmark\Performance\Dense\benchmark_cpu_%d_percentage_%d_wrokers_%d.csv`, CPU_COUNT, CPU_PERCENTAGE, POOL_SIZE))
	defer f.Close()

	if err != nil {
		log.Fatalln("failed to open file", err)
	}

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, record := range records {
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}

}
