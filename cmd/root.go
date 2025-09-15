package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rag-chatbot",
	Short: "RAG Chatbot POC",
}

var (
	configFile string
	loadKB     bool
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "configs/app-config.yaml", "application config file")
	rootCmd.PersistentFlags().BoolVar(&loadKB, "load-kb", false, "auto load knowledge base")
}
