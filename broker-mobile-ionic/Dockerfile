FROM nginx:1.17.10-alpine
EXPOSE 80
COPY ./www /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
RUN chmod -R 755 /usr/share/nginx/html