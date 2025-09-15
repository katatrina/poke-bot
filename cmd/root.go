package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-rag-chatbot",
	Short: "A RAG chatbot server built with Go",
	Long: `A Retrieval-Augmented Generation chatbot server that can ingest documents,
create embeddings, and provide intelligent responses using vector similarity search.`,
}

func Execute() error {
	return rootCmd.Execute()
}
