# EC2 Spot Instance Finder

EC2 Spot Instance Finder is a static website that helps you find the best deals on AWS EC2 Spot Instances across all regions. It provides an easy-to-use interface for comparing spot instance prices and finding the most cost-effective options for your needs.

## Features

- View the top 5 global deals across all AWS regions
- Search for the best deals in specific regions
- Automatically updated data every hour
- Comparison based on price per vCPU
- Easy-to-read table format for quick comparisons

## How It Works

1. A GitHub Action runs every hour to fetch the latest EC2 Spot Instance data.
2. The data is processed to find the best deals globally and per region.
3. The results are saved in a JSON file (`spot_data.json`).
4. The static website reads this JSON file to display the latest data.
5. Users can view global top deals or select a specific region to see the best deals there.

## Setup

To set up your own instance of the EC2 Spot Instance Finder:

1. Fork this repository.
2. Enable GitHub Pages in your forked repository:
   - Go to Settings > Pages
   - Set the source to your main branch
   - Save the changes
3. Set up the GitHub Action:
   - Go to Settings > Secrets and variables > Actions
   - Add a new repository secret named `PAT` with your GitHub Personal Access Token
4. The GitHub Action will now run automatically every hour, updating the spot instance data.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Data provided by [EC2 Spot Pricing API](https://aws.amazon.com/ec2/spot/pricing/)
- Inspired by the need for an easy way to compare EC2 Spot Instance prices across regions

## Disclaimer

This tool is for informational purposes only. Spot Instance prices are highly volatile and can change rapidly. Always verify the current pricing on the official AWS pricing page before making any decisions.
