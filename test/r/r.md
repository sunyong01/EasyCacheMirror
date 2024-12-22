## 测试 R

```bash
# 使用基础 R 镜像
docker run -it --rm r-base:latest bash -c "R -e '\
  options(repos = c(CRAN = \"http://192.168.0.124:8080/r\")); \
  install.packages(\"ggplot2\")'"

# 使用 RStudio
docker run -it --rm rocker/rstudio:latest bash -c "R -e '\
  options(repos = c(CRAN = \"http://192.168.0.124:8080/r\")); \
  install.packages(c(\"tidyverse\", \"devtools\"))'"
``` 