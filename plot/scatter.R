#!/usr/bin/env Rscript
depPackages = c("ggplot2", "hrbrthemes", "plotly", "htmlwidgets", "tidyr", "ggdendro", "reshape2", "grid", "heatmaply")
## Check load available dependencies, else install & load
package.check <- lapply(
  depPackages,
  FUN = function(x) {
    if (!require(x, character.only = TRUE)) {
      install.packages(x, dependencies = TRUE)
      library(x, character.only = TRUE)
    }
  }
)

data <- read.csv("pc.csv")

p <- ggplot(data, aes(x=pc1, y=pc2, color=data$chr3.165966638.AA.)) +
  geom_point() +
  scale_color_gradient2(low="white", mid="blue", high="red", limits = c(0, 1), oob = scales::squish)

p