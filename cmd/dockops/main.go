package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/client"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/moby/moby/pkg/stdcopy"
	"github.com/spf13/cobra"
)

var (
	dockerClient *client.Client
	// Flags para el comando start
	containerPort string
	volumePath    string
	command       string
)

func init() {
	rand.Seed(time.Now().UnixNano())
	var err error
	dockerClient, err = client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inicializando el cliente Docker: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "docker-cli",
		Short: "Una CLI minimalista para Docker usando Moby",
	}

	var startCmd = &cobra.Command{
		Use:   "start [imagen]",
		Short: "Inicia un contenedor de una imagen",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := startImage(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Agregar flags al comando start
	startCmd.Flags().StringVarP(&containerPort, "port", "p", "", "Puerto para exponer (formato: host:container)")
	startCmd.Flags().StringVarP(&volumePath, "volume", "v", "", "Volumen para montar (formato: host:container)")
	startCmd.Flags().StringVarP(&command, "cmd", "c", "/bin/sh", "Comando para ejecutar")

	var stopCmd = &cobra.Command{
		Use:   "stop [containerId]",
		Short: "Detiene un contenedor por ID o nombre",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := stopContainer(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var logsCmd = &cobra.Command{
		Use:   "logs [containerId]",
		Short: "Muestra los logs de un contenedor",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := showLogs(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var removeCmd = &cobra.Command{
		Use:   "remove [imageId]",
		Short: "Elimina una imagen por su ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := removeImage(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "Lista todas las imágenes disponibles",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listImages(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(startCmd, stopCmd, logsCmd, removeCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startImage(imageName string) error {
	ctx := context.Background()

	fmt.Printf("Descargando imagen %s...\n", imageName)
	reader, err := dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("error al descargar la imagen: %v", err)
	}
	defer reader.Close()

	// Mostrar progreso de descarga
	dec := json.NewDecoder(reader)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error al decodificar el progreso: %v", err)
		}

		if event.Error != nil {
			return fmt.Errorf("error en la descarga: %s", event.Error.Message)
		}

		if event.Progress != nil {
			fmt.Printf("\r%s: %s", event.Status, event.Progress.String())
		} else {
			fmt.Println(event.Status)
		}
	}

	// Configuración del contenedor
	containerName := generateRandomContainerName()
	config := &container.Config{
		Image: imageName,
		Cmd:   strings.Split(command, " "),
		Tty:   true,
	}

	// Configuración del host
	hostConfig := &container.HostConfig{}

	// Configurar puerto si se especificó
	if containerPort != "" {
		parts := strings.Split(containerPort, ":")
		if len(parts) != 2 {
			return fmt.Errorf("formato de puerto inválido. Use host:container")
		}
		hostConfig.PortBindings = nat.PortMap{
			nat.Port(parts[1] + "/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: parts[0]}},
		}
	}

	// Configurar volumen si se especificó
	if volumePath != "" {
		parts := strings.Split(volumePath, ":")
		if len(parts) != 2 {
			return fmt.Errorf("formato de volumen inválido. Use host:container")
		}
		hostConfig.Binds = []string{volumePath}
	}

	fmt.Printf("\nCreando contenedor con imagen %s...\n", imageName)
	resp, err := dockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("error al crear el contenedor: %v", err)
	}

	handleInterrupt(resp.ID)

	fmt.Printf("Iniciando contenedor %s...\n", resp.ID[:12])
	if err := dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error al iniciar el contenedor: %v", err)
	}

	fmt.Printf("Contenedor iniciado exitosamente (nombre: %s, ID: %s)\n", containerName, resp.ID[:12])
	return nil
}

func stopContainer(containerID string) error {
	ctx := context.Background()

	fmt.Printf("Deteniendo contenedor %s...\n", containerID)
	if err := dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("error al detener el contenedor: %v", err)
	}

	fmt.Println("Contenedor detenido exitosamente")
	return nil
}

func showLogs(containerID string) error {
	ctx := context.Background()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	logsReader, err := dockerClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("error al obtener logs: %v", err)
	}
	defer logsReader.Close()

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, logsReader)
	return err
}

func removeImage(imageID string) error {
	ctx := context.Background()

	options := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	_, err := dockerClient.ImageRemove(ctx, imageID, options)
	if err != nil {
		return fmt.Errorf("error al eliminar la imagen: %v", err)
	}

	fmt.Printf("Imagen %s eliminada exitosamente\n", imageID)
	return nil
}

func listImages() error {
	ctx := context.Background()

	images, err := dockerClient.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("error al listar imágenes: %v", err)
	}

	fmt.Println("Imágenes disponibles:")
	fmt.Printf("%-12s %-50s %-20s\n", "IMAGE ID", "REPOSITORY", "TAG")
	for _, image := range images {
		for _, tag := range image.RepoTags {
			repo, tagName := parseRepoTag(tag)
			fmt.Printf("%-12s %-50s %-20s\n", image.ID[:12], repo, tagName)
		}
		if len(image.RepoTags) == 0 {
			fmt.Printf("%-12s %-50s %-20s\n", image.ID[:12], "<none>", "<none>")
		}
	}

	return nil
}

// Funciones auxiliares
func generateRandomContainerName() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return "docker-cli-" + string(b)
}

func handleInterrupt(containerID string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nInterrumpido. Deteniendo el contenedor...")
		ctx := context.Background()
		if err := dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			fmt.Fprintf(os.Stderr, "Error al detener el contenedor: %v\n", err)
		}
		os.Exit(0)
	}()
}

func parseRepoTag(repoTag string) (string, string) {
	parts := strings.Split(repoTag, ":")
	if len(parts) != 2 {
		return repoTag, "latest"
	}
	return parts[0], parts[1]
}
