## 测试 RubyGems

```bash
# 使用 gem
docker run -it --rm ruby:latest bash -c "\
  gem sources --add http://192.168.0.124:8080/rubygems --remove https://rubygems.org/ && \
  gem install rails"

# 使用 bundler
docker run -it --rm ruby:latest bash -c "\
  bundle config mirror.https://rubygems.org http://192.168.0.124:8080/rubygems && \
  mkdir test-project && cd test-project && \
  echo 'source \"http://192.168.0.124:8080/rubygems\"' > Gemfile && \
  echo 'gem \"rails\"' >> Gemfile && \
  bundle install"
``` 