<template>
  <div class="wheel_wrap">
    <h1>Wheel of Fortune</h1>
    <Wheel
      ref="wheel"
      :gift="gift"
      :data="data"
      @done="done"
      :imgParams="logo"
      @click="spinTheWheel"
    />

    <div v-if="this.spun">
    User has won {{ this.winnings.value }}
    </div>
  </div>
</template>

<script>
import { Wheel } from "vue3-fortune-wheel";
import 'vue3-fortune-wheel/dist/library.css'

import { WheelClient } from "./services/api_grpc_web_pb.js"
import { SpinWheelReq } from "./services/api_pb.js"

let wheelservice = new WheelClient("http://localhost:8080", null, null);
let wheelName = "test"
export default {
  name: "App",
  components: {
    Wheel,
  },
  created() {
     let req = new SpinWheelReq();
      req.setName(wheelName);

      wheelservice.spinWheel(req, {}, (error, response) => {
        this.gift = response.toObject().winningsegmentindex;
      });
  },
  data() {
    return {
      gift: null,
      logo: {
        width: 100,
        height: 120,
        src:
          "https://upload.wikimedia.org/wikipedia/commons/thumb/9/95/Vue.js_Logo_2.svg/2367px-Vue.js_Logo_2.svg.png",
      },
      winnings: {},
      spun: false,
      spinning: false,
      data: [
        {
          id: 1,
          value: "Gift 1",
          bgColor: "#7d7db3",
          color: "#ffffff",
        },
        {
          id: 2,
          value: "Gift 2",
          bgColor: "#ffffff",
          color: "#000000",
        },
        {
          id: 3,
          value: "Gift 3",
          bgColor: "#c92729",
          color: "#ffffff",
        },
        {
          id: 4,
          value: "Gift 4",
          bgColor: "#7d7db3",
          color: "#ffffff",
        },
        {
          id: 5,
          value: "Gift 5",
          bgColor: "#ffffff",
          color: "#000000",
        },
        {
          id: 6,
          value: "Gift 6",
          bgColor: "#c92729",
          color: "#ffffff",
        },
      ],
    };
  },
  methods: {
    done(params) {
      this.spinning = false;
      this.winnings = params;
      this.spun = true;
      this.$refs.wheel.clicked = false;
    },
    spinTheWheel() {
      if (this.spinning) {
        return
      }
      this.spun = false;
      this.spinning = true;

      let req = new SpinWheelReq();
      req.setName(wheelName);

      wheelservice.spinWheel(req, {}, (error, response) => {
        this.gift = response.toObject().winningsegmentindex;
      });
      this.$refs.wheel.spin();
    },
  },
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}
.wheel_wrap {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}
</style>
