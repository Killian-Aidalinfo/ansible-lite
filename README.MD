## Installation source

```bash
sudo curl -fsSL https://aidalinfo.github.io/aidalinfo-source/aidalinfo.gpg.key | sudo tee /etc/apt/trusted.gpg.d/aidalinfo.asc \
echo "deb [signed-by=/etc/apt/trusted.gpg.d/aidalinfo.asc] https://aidalinfo.github.io/aidalinfo-source/ stable main" | sudo tee /etc/apt/sources.list.d/aidalinfo.list \
sudo apt-get update
sudo apt-get install ansible-lite

```