# Download all dependencies
go mod download

# Create env file
JWT_SECRET=$(openssl rand -base64 64)
POLKA_KEY=f271c81ff7084ee5b99a5091b42d486e # This is a dummy key, so no need to worry

touch .env

echo "JWT_SECRET=${JWT_SECRET}" >> .env
echo "POLKA_KEY=${POLKA_KEY}" >> .env

echo "# Setup Complete"