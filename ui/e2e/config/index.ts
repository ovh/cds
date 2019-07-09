export default {
    environment: process.env.NODE_ENV || 'development',
    baseUrl: process.env.BASE_URL ||  'http://localhost:4200',
    username: process.env.CDS_USERNAME,
    password: process.env.CDS_PASSWORD
}
