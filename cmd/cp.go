package cmd

import (
	"errors"
	"fmt"
	"github.com/oxio/rediscp/package/rediscp"

	"github.com/spf13/cobra"
)

func NewCpCmd() *cobra.Command {
	var pattern string
	var verbose, ignoreTTL, skipExisting, replaceExisting bool

	cmd := &cobra.Command{
		Use:   "rediscp <source-addr> <target-addr>",
		Short: "Copies keys from one Redis instance to another",
		Long: `Copies keys from one Redis instance to another.

The source and target addresses should be specified in the format redis://[:password]@host:port[/db].

Example:
rediscp redis://:srcpass@localhost:6379/0 redis://:targetpass@localhost:6380/1`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 2 {
				return errors.New("source-addr and target-addr are required positional arguments")
			}
			sourceUrl := args[0]
			targetUrl := args[1]
			ctx := cmd.Context()

			// Create Redis clients for source and target Redis instances
			sourceClient := rediscp.NewRedisClient(sourceUrl)
			targetClient := rediscp.NewRedisClient(targetUrl)

			scanner := &rediscp.RedisScanner{}

			copier := &rediscp.RedisCopier{
				IgnoreTTL:       ignoreTTL,
				SkipExisting:    skipExisting,
				ReplaceExisting: replaceExisting,
				Verbose:         verbose,
			}

			// Get keys from source Redis
			keys, err := scanner.ScanKeys(ctx, sourceClient, pattern)
			if err != nil {
				return err
			}

			// Copy the keys from source Redis to target Redis
			copied, skipped, err := copier.CopyKeys(ctx, sourceClient, targetClient, keys)

			fmt.Printf("Copied total of %d keys, skipped %d keys\n", copied, skipped)

			if err != nil {
				var e *rediscp.ErrBusykey
				if errors.As(err, &e) {
					return fmt.Errorf(
						"key \"%s\" already exists in target Redis. "+
							"Use one of the following flags to bypass this error: "+
							"--skip-existing or --replace-existing\n",
						e.Key,
					)
				}
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&pattern, "keys", "k", "*", "Keys pattern to copy eg. \"foo:*\"")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (default: false)")
	cmd.Flags().BoolVarP(&skipExisting, "skip-existing", "s", false, "Weather or not to skip already existing keys in the target Redis (ignoring BUSYKEY error) (default: false)")
	cmd.Flags().BoolVarP(&replaceExisting, "replace-existing", "r", false, "Weather or not to replace already existing keys in the target Redis (default: false)")
	cmd.Flags().BoolVar(&ignoreTTL, "ignore-ttl", false, "Weather or not to ignore TTL of keys (default: false)")

	return cmd
}
