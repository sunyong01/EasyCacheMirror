## 测试 Conda

```bash
# 使用 conda
docker run -it --rm continuumio/miniconda3 bash -c "\
  conda config --add channels http://192.168.0.124:8080/conda/pkgs/main && \
  conda config --set show_channel_urls yes && \
  conda install numpy pandas"

# 使用 mamba（更快的 conda 替代品）
docker run -it --rm continuumio/miniconda3 bash -c "\
  conda install -y mamba && \
  mamba config --add channels http://192.168.0.124:8080/conda/pkgs/main && \
  mamba config --set show_channel_urls yes && \
  mamba install scipy scikit-learn"
```

## 测试特定渠道
这一点暂未想好怎么实现。根据清华源的做法
```yaml
custom_channels:
  conda-forge: https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud
  pytorch: https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud

```
暂时这样做
```bash
# 测试 conda-forge 渠道
docker run -it --rm continuumio/miniconda3 bash -c "\
  conda config --add channels http://192.168.0.124:8080/conda/pkgs/main && \
  conda config --set custom_channels.auto http://192.168.0.124:8080/conda/cloud && \
  conda install jupyterlab"

# 测试 pytorch 渠道
docker run -it --rm continuumio/miniconda3 bash -c "\
  conda config --add channels http://192.168.0.124:8080/conda/pkgs/main && \
  conda config --set custom_channels.auto http://192.168.0.124:8080/conda/cloud && \
  conda install pytorch torchvision"
``` 