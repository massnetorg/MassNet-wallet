package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/cmd/masswalletcli/utils"
	"massnet.org/mass-wallet/logging"
)

var createCertCmd = &cobra.Command{
	Use:   "createcert <directory>",
	Short: "Creates a new PEM-encoded x.509 certificate and writes to directory.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		fi, err := os.Stat(args[0])
		if os.IsNotExist(err) || !fi.IsDir() {
			return fmt.Errorf("directory not exists: %s", args[0])
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cert := filepath.Join(args[0], "cert.crt")
		key := filepath.Join(args[0], "cert.key")
		err := utils.GenerateTLSCertPair(cert, key)
		if err != nil {
			logging.VPrint(logging.ERROR, "gencertpair error", logging.LogFormat{
				"directory": args[0],
				"err":       err,
			})
			return err
		}
		jww.FEEDBACK.Println(cert, "done")
		jww.FEEDBACK.Println(key, "done")
		return nil
	},
}

var getClientStatusCmd = &cobra.Command{
	Use:   "getclientstatus",
	Short: "Returns data about connected client.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getclientstatus called", EmptyLogFormat)

		resp := &pb.GetClientStatusResponse{}
		return ClientCall("/v1/client/status", GET, nil, resp)
	},
}

var getBestBlockCmd = &cobra.Command{
	Use:   "getbestblock",
	Short: "Returns data about best block.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getbestblock called", EmptyLogFormat)

		resp := &pb.GetBestBlockResponse{}
		return ClientCall("/v1/blocks/best", GET, nil, resp)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop wallet node.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "stop called", EmptyLogFormat)

		resp := &pb.QuitClientResponse{}
		return ClientCall("/v1/client/quit", POST, nil, resp)
	},
}
